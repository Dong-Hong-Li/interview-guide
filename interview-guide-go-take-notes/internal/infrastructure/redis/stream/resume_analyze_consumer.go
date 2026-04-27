package redisstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/infrastructure/ai"
	"interview-guide-go/internal/infrastructure/ai/promptprofile"
	"interview-guide-go/shared/logmsg"
	sharedresume "interview-guide-go/shared/resume"
	"interview-guide-go/shared/streamkey"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 同时最多在跑的简历分析协程数：用缓冲 chan 作「座位」，坐满后新任务在 <-sem 处阻塞，避免无限制开协程。
const resumeAnalyzeMaxConcurrent = 10

// StartResumeAnalyzeConsumer 启动简历分析消费者：XREADGROUP 读流，调用 grader.Grade，经 ResumeWriter 写库。
func StartResumeAnalyzeConsumer(ctx context.Context, rdb *redis.Client, w repository.ResumeWriter, grader *ai.ResumeGrader, lg *zap.Logger) {
	if rdb == nil || w == nil || grader == nil {
		return
	}
	consumer := fmt.Sprintf("analyze-consumer-go-%d", os.Getpid()) // 生成消费者名称
	go runResumeAnalyzeConsumer(ctx, rdb, w, grader, lg, consumer)
}

// 运行简历分析消费者
func runResumeAnalyzeConsumer(ctx context.Context, rdb *redis.Client, w repository.ResumeWriter, grader *ai.ResumeGrader, lg *zap.Logger, consumer string) {
	// 创建消费者组
	if err := ensureResumeAnalyzeGroup(ctx, rdb); err != nil {
		lg.Error(logmsg.MsgResumeAnalyzeCreateConsumerGroup, zap.Error(err))
		return
	}

	// sem：容量 = 座位数；先发 token 进 sem 再 go，处理完 <-sem 还座，限并发不藏 worker 池
	sem := make(chan struct{}, resumeAnalyzeMaxConcurrent)
	lg.Info(logmsg.MsgResumeAnalyzeConsumerStarted,
		zap.String(logmsg.FieldConsumer, consumer),
		zap.Int("maxConcurrent", resumeAnalyzeMaxConcurrent),
	)

	/*
		消费者组创建成功后,开始读取简历分析队列消息,如果未读到消息,则阻塞3秒
		这里本质上是在无线循环读取简历分析队列消息,不管读到消息与否,都会继续读取
		如果读到消息,则遍历消息,处理每条简历分析队列消息
	*/
	for {
		select {
		case <-ctx.Done():
			lg.Info(logmsg.MsgResumeAnalyzeConsumerStopped)
			return
		default:
		}

		res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    streamkey.GroupResumeAnalyze,
			Consumer: consumer,
			Streams:  []string{streamkey.StreamResumeAnalyze, ">"},
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
				if recErr := ensureResumeAnalyzeGroup(ctx, rdb); recErr != nil {
					lg.Warn(logmsg.MsgResumeAnalyzeXReadGroup, zap.Error(recErr))
					time.Sleep(time.Second)
				}
				continue
			}
			lg.Warn(logmsg.MsgResumeAnalyzeXReadGroup, zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				// 拿座位（满员则阻塞）；退出时让出等待
				select {
				case <-ctx.Done():
					lg.Info(logmsg.MsgResumeAnalyzeConsumerStopped)
					return
				case sem <- struct{}{}:
				}
				m := msg
				go func() {
					defer func() { <-sem }()
					processResumeAnalyzeMessage(ctx, rdb, w, grader, lg, m)
				}()
			}
		}
	}
}

// 确保简历分析消费者组存在（MKSTREAM：无 key 时创建空流）。 返回错误则表示失败，BUSYGROUP 视为已存在，返回 nil。
func ensureResumeAnalyzeGroup(ctx context.Context, rdb *redis.Client) error {
	err := rdb.XGroupCreateMkStream(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") { // BUSYGROUP 视为已存在，返回 nil。
		return err
	}
	return nil
}

// 处理简历分析队列消息
// ctx 上下文
// rdb Redis 客户端
// w ResumeWriter 简历写入器
// grader ResumeGrader 简历分析器
// lg 日志记录器
// msg 简历分析队列消息
func processResumeAnalyzeMessage(ctx context.Context, rdb *redis.Client, resumeWriter repository.ResumeWriter, grader *ai.ResumeGrader, lg *zap.Logger, msg redis.XMessage) {
	// 1 获取简历ID和内容
	resumeIDStr, _ := msg.Values[streamkey.StreamFieldResumeID].(string)
	if resumeIDStr == "" {
		resumeIDStr = fmt.Sprint(msg.Values[streamkey.StreamFieldResumeID])
	}
	resumeText, _ := msg.Values[streamkey.StreamFieldContent].(string)
	if resumeIDStr == "" || resumeText == "" { // 如果简历ID或内容为空,触发跳过无效消息
		lg.Warn(logmsg.MsgResumeAnalyzeSkipBadMessage, zap.String(logmsg.FieldID, msg.ID))
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}

	resumeID, err := strconv.ParseInt(resumeIDStr, 10, 64)
	if err != nil { // 如果简历ID无法解析,触发跳过无效消息
		lg.Warn(logmsg.MsgResumeAnalyzeBadResumeID, zap.String(logmsg.FieldID, msg.ID))
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}

	// 2 获取简历信息
	rec, err := resumeWriter.GetResumeForAnalyze(ctx, resumeID)
	if err != nil { // 如果简历不存在,触发跳过无效消息
		if errors.Is(err, repository.ErrResumeNotFound) {
			lg.Warn(logmsg.MsgResumeAnalyzeResumeGone, zap.Int64(logmsg.FieldResumeID, resumeID))
			_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
			return
		}
		lg.Warn(logmsg.MsgResumeAnalyzeLoadResume, zap.Error(err))
		return
	}

	// 3 获取面试官角色
	interviewerRole, _ := promptprofile.Parse(strings.TrimSpace(rec.InterviewerRole))

	lg.Info(logmsg.MsgResumeAnalyzeAITaskReceived,
		zap.Int64(logmsg.FieldResumeID, resumeID),
		zap.String(logmsg.FieldID, msg.ID),
		zap.String(logmsg.FieldOriginalFilename, strings.TrimSpace(rec.OriginalFilename)),
		zap.String(logmsg.FieldInterviewerRole, interviewerRole),
	)

	// 4 异步更新简历分析状态为处理中
	if err := resumeWriter.UpdateAnalyzeStatus(ctx, resumeID, string(sharedresume.AnalyzeStatusProcessing), ""); err != nil { // 如果更新失败,触发跳过无效消息
		lg.Warn(logmsg.MsgResumeAnalyzeMarkProcessing, zap.Error(err))
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}
	lg.Info(logmsg.MsgResumeAnalyzeAIBeginGrade,
		zap.Int64(logmsg.FieldResumeID, resumeID),
		zap.Int(logmsg.FieldRuneCount, utf8.RuneCountInString(resumeText)),
		zap.String(logmsg.FieldInterviewerRole, interviewerRole),
	)

	// 5 开始调用模型评分
	actx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	aiStart := time.Now()
	// 调用模型评分 scores 评分结果
	scores, err := grader.Grade(actx, resumeText, interviewerRole)
	cancel()
	// 计算评分总耗时
	gradeElapsed := time.Since(aiStart)
	if err != nil {
		lg.Warn(logmsg.MsgResumeAnalyzeGradeFailed, zap.Int64(logmsg.FieldResumeID, resumeID), zap.Duration(logmsg.FieldAIDuration, gradeElapsed), zap.Error(err))
		_ = resumeWriter.UpdateAnalyzeStatus(ctx, resumeID, string(sharedresume.AnalyzeStatusFailed), err.Error())
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}
	lg.Info(logmsg.MsgResumeAnalyzeAIGradeOK,
		zap.Int64(logmsg.FieldResumeID, resumeID),
		zap.Int(logmsg.FieldOverallScore, scores.Overall),
		zap.Duration(logmsg.FieldAIDuration, gradeElapsed),
	)

	// 6 序列化优势项 strengthsJSON 优势项序列化后的 JSON
	strengthsJSON, err := json.Marshal(scores.Strengths)
	if err != nil {
		lg.Warn(logmsg.MsgResumeAnalyzeMarshalStrengths, zap.Int64(logmsg.FieldResumeID, resumeID), zap.Duration(logmsg.FieldAIDuration, gradeElapsed), zap.Error(err))
		_ = resumeWriter.UpdateAnalyzeStatus(ctx, resumeID, string(sharedresume.AnalyzeStatusFailed), sharedresume.FailedMarshalStrengthsMsgPrefix+err.Error())
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}

	// 7 创建简历分析结果行
	row := &repository.ResumeAnalysisInput{
		ResumeID:        resumeID,
		OverallScore:    intPtr(scores.Overall),
		ContentScore:    intPtr(scores.Content),
		StructureScore:  intPtr(scores.Structure),
		SkillMatchScore: intPtr(scores.SkillMatch),
		ExpressionScore: intPtr(scores.Expression),
		ProjectScore:    intPtr(scores.Project),
		Summary:         scores.Summary,
		StrengthsJSON:   string(strengthsJSON),
		SuggestionsJSON: scores.SuggestionsJSON,
	}

	// 8 插入简历分析结果行
	if err := resumeWriter.InsertResumeAnalysis(ctx, row); err != nil {
		lg.Warn(logmsg.MsgResumeAnalyzeInsertAnalysis, zap.Duration(logmsg.FieldAIDuration, gradeElapsed), zap.Error(err))
		_ = resumeWriter.UpdateAnalyzeStatus(ctx, resumeID, string(sharedresume.AnalyzeStatusFailed), sharedresume.FailedSaveAnalysisMsgPrefix+err.Error())
		_ = rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err()
		return
	}

	// 9 更新简历分析状态为已完成
	if err := resumeWriter.UpdateAnalyzeStatus(ctx, resumeID, string(sharedresume.AnalyzeStatusCompleted), ""); err != nil {
		lg.Warn(logmsg.MsgResumeAnalyzeMarkCompleted, zap.Duration(logmsg.FieldAIDuration, gradeElapsed), zap.Error(err))
	}

	// 10 确认消息
	if err := rdb.XAck(ctx, streamkey.StreamResumeAnalyze, streamkey.GroupResumeAnalyze, msg.ID).Err(); err != nil {
		lg.Warn(logmsg.MsgResumeAnalyzeXAck, zap.Error(err))
	}
	lg.Info(logmsg.MsgResumeAnalyzeDone, zap.Int64(logmsg.FieldResumeID, resumeID), zap.Duration(logmsg.FieldAIDuration, gradeElapsed))
}

func intPtr(v int) *int {
	return &v
}
