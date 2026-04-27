package redisstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"interview-guide-go/internal/application/interview/service"
	domainiv "interview-guide-go/internal/domain/interview"
	"interview-guide-go/internal/infrastructure/ai"
	"interview-guide-go/internal/infrastructure/ai/promptprofile"
	"interview-guide-go/internal/interfaces/api/dto"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/streamkey"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 将错误转换为面试评估错误消息
func interviewEvalErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errmsg.InterviewEvalTimeout
	}
	if errors.Is(err, context.Canceled) {
		return errmsg.InterviewQuestionGenCanceled
	}
	s := strings.TrimSpace(err.Error())
	ls := strings.ToLower(s)
	if strings.Contains(ls, "deadline exceeded") || strings.Contains(ls, "context deadline exceeded") || strings.Contains(ls, "client.timeout") {
		return errmsg.InterviewEvalTimeout
	}
	if s == "" {
		return errmsg.InterviewEvalTimeout
	}
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

// 检查并确保面试评估消费者组存在
func ensureInterviewEvaluateGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, streamkey.StreamInterviewEvaluate, streamkey.GroupInterviewEvaluate, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// StartInterviewEvaluateConsumer 启动面试 LLM 评估消费者；写库经 EvaluateProcessor + InterviewMapper。
func StartInterviewEvaluateConsumer(ctx context.Context, rdb *redis.Client, proc *service.EvaluateProcessor, eval *ai.InterviewEvaluator, lg *zap.Logger) {
	if rdb == nil || proc == nil || eval == nil {
		return
	}
	consumer := fmt.Sprintf("evaluate-consumer-go-%d", os.Getpid())
	go runInterviewEvaluateConsumer(ctx, rdb, proc, eval, lg, consumer)
}

// 运行面试 LLM 评估消费者
func runInterviewEvaluateConsumer(ctx context.Context, rdb *redis.Client, proc *service.EvaluateProcessor, eval *ai.InterviewEvaluator, lg *zap.Logger, consumer string) {
	// 确保面试评估消费者组存在
	if err := ensureInterviewEvaluateGroup(ctx, rdb); err != nil {
		lg.Error(logmsg.MsgInterviewEvaluateCreateConsumerGroup, zap.Error(err))
		return
	}
	lg.Info(logmsg.MsgInterviewEvaluateConsumerStarted, zap.String(logmsg.FieldConsumer, consumer))

	for {
		select {
		case <-ctx.Done():
			lg.Info(logmsg.MsgInterviewEvaluateConsumerStopped)
			return
		default:
		}

		res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    streamkey.GroupInterviewEvaluate,
			Consumer: consumer,
			Streams:  []string{streamkey.StreamInterviewEvaluate, ">"},
			Count:    4,
			Block:    3 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil || err == context.Canceled {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			if strings.Contains(strings.ToUpper(err.Error()), "NOGROUP") {
				if recErr := ensureInterviewEvaluateGroup(ctx, rdb); recErr != nil {
					lg.Warn(logmsg.MsgInterviewEvaluateXReadGroup, zap.Error(recErr))
					time.Sleep(time.Second)
				}
				continue
			}
			lg.Warn(logmsg.MsgInterviewEvaluateXReadGroup, zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				// 处理面试 LLM 评估消息
				processInterviewEvaluateMessage(ctx, rdb, proc, eval, lg, msg)
			}
		}
	}
}

// 处理面试 LLM 评估消息
func processInterviewEvaluateMessage(ctx context.Context, rdb *redis.Client, proc *service.EvaluateProcessor, eval *ai.InterviewEvaluator, lg *zap.Logger, msg redis.XMessage) {
	// 确认消息 ACK : 处理成功（或决定永久丢弃无效消息）后，要调 XACK，告诉 Redis：这条我已经处理完了，可以从 PEL 里删掉。
	ack := func() {
		_ = rdb.XAck(ctx, streamkey.StreamInterviewEvaluate, streamkey.GroupInterviewEvaluate, msg.ID).Err()
	}

	// 获取会话ID
	sid, _ := msg.Values[streamkey.StreamFieldEvalSessionID].(string)
	if strings.TrimSpace(sid) == "" {
		sid = fmt.Sprint(msg.Values[streamkey.StreamFieldEvalSessionID])
	}

	// 清理会话ID
	sid = strings.TrimSpace(sid)
	if sid == "" {
		lg.Warn(logmsg.MsgInterviewEvaluateSkipBadMessage, zap.String(logmsg.FieldID, msg.ID))
		ack()
		return
	}

	// 加载面试会话信息
	sess, err := proc.WorkerGetSessionByPublicID(ctx, sid)
	if err != nil || sess == nil {
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果会话不存在，记录错误并返回
			lg.Warn(logmsg.MsgInterviewEvaluateSessionGone, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
			return
		}
		lg.Info(logmsg.MsgInterviewEvaluateSessionGone, zap.String(logmsg.FieldSessionID, sid))
		ack()
		return
	}

	// 获取会话状态
	st := strings.ToUpper(strings.TrimSpace(sess.Status))
	// 如果会话状态为已评估，则跳过
	if st == domainiv.InterviewStatusEvaluated {
		// 获取评估状态
		es := strings.ToUpper(strings.TrimSpace(sess.EvaluateStatus))
		if es == domainiv.InterviewEvaluateStatusCompleted {
			ack()
			return
		}
	}

	// 如果会话状态不为已完成，则跳过 (状态异常)
	if st != domainiv.InterviewStatusCompleted {
		// 记录日志
		lg.Info(logmsg.MsgInterviewEvaluateNotReady, zap.String(logmsg.FieldSessionID, sid), zap.String(logmsg.FieldStatus, st))
		ack()
		return
	}

	// 获取评估状态 不为待处理，则跳过 (状态异常)
	evalSt := strings.ToUpper(strings.TrimSpace(sess.EvaluateStatus))
	// 如果评估状态不为待处理，则跳过
	if evalSt != domainiv.InterviewEvaluateStatusPending {
		// 如果评估状态为已完成且会话状态为已评估，则跳过
		if evalSt == domainiv.InterviewEvaluateStatusCompleted && st == domainiv.InterviewStatusEvaluated {
			ack()
			return
		}
		lg.Info(logmsg.MsgInterviewEvaluateNotReady, zap.String(logmsg.FieldSessionID, sid), zap.String("evaluateStatus", evalSt))
		ack()
		return
	}

	// 尝试标记评估状态为处理中
	marked, err := proc.WorkerTryMarkEvaluateProcessing(ctx, sess.ID)
	// 如果标记失败，则记录日志并返回
	if err != nil {
		lg.Warn(logmsg.MsgInterviewEvaluateMarkProcessing, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
		return
	}

	// 如果标记失败，则加载会话信息并检查会话状态
	if !marked {
		s2, e2 := proc.WorkerGetSessionByPublicID(ctx, sid)
		if e2 == nil && s2 != nil {
			st2 := strings.ToUpper(strings.TrimSpace(s2.Status))
			es2 := strings.ToUpper(strings.TrimSpace(s2.EvaluateStatus))
			if st2 == domainiv.InterviewStatusEvaluated && es2 == domainiv.InterviewEvaluateStatusCompleted {
				ack()
				return
			}
		}
		ack()
		return
	}

	// 加载题目
	var qs []dto.InterviewQuestion
	// 如果题目 JSON 不为空，则解析题目
	if raw := strings.TrimSpace(sess.QuestionsJSON); raw != "" {
		// 如果解析失败，则标记评估失败并记录日志
		if err := json.Unmarshal([]byte(raw), &qs); err != nil {
			_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, "题目 JSON 解析失败: "+err.Error())
			lg.Warn(logmsg.MsgInterviewEvaluateLLMFailed, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
			ack()
			return
		}
	}

	// 如果题目为空，则标记评估失败并记录日志
	if len(qs) == 0 {
		_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, "会话无题目，无法评估")
		ack()
		return
	}

	// 加载答案
	answers, err := proc.WorkerListInterviewAnswers(ctx, sess.ID)
	// 如果加载答案失败，则标记评估失败并记录日志
	if err != nil {
		_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, "加载答案失败: "+err.Error())
		lg.Warn(logmsg.MsgInterviewEvaluateLLMFailed, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
		ack()
		return
	}

	// 合并题目答案
	for _, a := range answers {
		if a.QuestionIndex >= 0 && a.QuestionIndex < len(qs) {
			qs[a.QuestionIndex].UserAnswer = a.UserAnswer
		}
	}

	// 加载简历正文和面试官角色
	resumeText, interviewerRoleRaw, err := proc.WorkerGetResumeTextAndInterviewerRole(ctx, sess.ResumeID)
	// 如果加载简历正文和面试官角色失败，则标记评估失败并记录日志
	if err != nil {
		_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, "简历不存在，无法评估")
		lg.Warn(logmsg.MsgInterviewEvaluateLLMFailed, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
		ack()
		return
	}

	// 解析面试官角色
	interviewerRole, _ := promptprofile.Parse(strings.TrimSpace(interviewerRoleRaw))

	lg.Info(logmsg.MsgInterviewEvaluateAIBegin,
		zap.String(logmsg.FieldSessionID, sid),
		zap.String(logmsg.FieldInterviewerRole, interviewerRole),
		zap.Int("questionCount", len(qs)),
	)

	actx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	aiStart := time.Now()

	// 评估面试 (对整场面试打分并生成 dto.InterviewReport)
	report, evalErr := eval.EvaluateInterview(actx, sid, resumeText, interviewerRole, qs)
	cancel() // 结束计时
	evalElapsed := time.Since(aiStart)

	// 如果评估失败，则标记评估失败并记录日志
	if evalErr != nil {
		_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, interviewEvalErrorMessage(evalErr))
		lg.Warn(logmsg.MsgInterviewEvaluateLLMFailed, zap.String(logmsg.FieldSessionID, sid), zap.Duration(logmsg.FieldAIDuration, evalElapsed), zap.Error(evalErr))
		ack()
		return
	}

	// 转换为 application 层模型
	appRep := evaluationReportFromDTO(&report)
	// 保存评估结果
	if err := proc.WorkerSaveEvaluationResult(ctx, sess.ID, appRep); err != nil {
		_ = proc.WorkerMarkEvaluateFailed(ctx, sess.ID, "保存报告: "+interviewEvalErrorMessage(err))
		lg.Warn(logmsg.MsgInterviewEvaluatePersistFailed, zap.String(logmsg.FieldSessionID, sid), zap.Duration(logmsg.FieldAIDuration, evalElapsed), zap.Error(err))
		ack()
		return
	}

	// 更新会话缓存
	proc.WorkerWarmSessionCacheAfterEvaluate(ctx, sid)
	lg.Info(logmsg.MsgInterviewEvaluateDone, zap.String(logmsg.FieldSessionID, sid), zap.Int(logmsg.FieldOverallScore, report.OverallScore), zap.Duration(logmsg.FieldAIDuration, evalElapsed))
	ack()
}
