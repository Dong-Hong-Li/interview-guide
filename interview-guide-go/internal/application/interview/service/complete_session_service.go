package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	domainiv "interview-guide-go/internal/domain/interview"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// CompleteSessionService POST /sessions/{sessionId}/complete：允许候选人在未答完全部题目时提前交卷。
type CompleteSessionService struct {
	sessions repository.InterviewSessionWriter
	cache    repository.InterviewSessionCache
	enqueue  repository.InterviewEvaluateEnqueuer
}

// NewCompleteSessionService 由 wire 注入。
func NewCompleteSessionService(
	sessions repository.InterviewSessionWriter,
	cache repository.InterviewSessionCache,
	enqueue repository.InterviewEvaluateEnqueuer,
) *CompleteSessionService {
	return &CompleteSessionService{sessions: sessions, cache: cache, enqueue: enqueue}
}

// CompleteSession 将会话置为 COMPLETED，evaluate_status=PENDING，并入队 LLM 评估；然后刷新 Redis 会话缓存。
func (s *CompleteSessionService) CompleteSession(ctx context.Context, sid string) (any, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview session not configured")
	}
	// 获取会话记录
	rec, err := s.sessions.GetSessionRecordForSubmit(ctx, sid)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.CompleteInterviewFailed)
	}
	// 如果会话记录为空，则返回错误
	if rec == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}

	// 解析题目
	var qs []results.InterviewQuestion
	if raw := strings.TrimSpace(rec.QuestionsJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &qs); err != nil {
			return nil, response.Err(http.StatusInternalServerError, errmsg.CompleteInterviewFailed)
		}
	}
	// 解析会话状态
	st := domainiv.ParseSessionStatus(rec.Status)
	// 校验是否允许提前交卷
	if gerr := domainiv.CompleteInterviewGate(st, len(qs)); gerr != nil {
		if errors.Is(gerr, domainiv.ErrCompleteAlreadyDone) {
			return nil, response.Err(http.StatusBadRequest, errmsg.CompleteInterviewAlreadyDone)
		}
		if errors.Is(gerr, domainiv.ErrAnswerQuestionsNotReady) {
			return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
		}
		return nil, response.Err(http.StatusBadRequest, gerr.Error())
	}

	// 顺序：先把会话标记 COMPLETED，再置 evaluate_status=PENDING，最后入队评估，避免消费者抢在状态写入前执行。
	if err := s.sessions.UpdateInterviewSessionProgress(ctx, rec.InternalID, rec.CurrentQuestionIndex, domainiv.InterviewStatusCompleted, true); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.CompleteInterviewFailed)
	}
	if err := s.sessions.UpdateInterviewSessionEvaluatePending(ctx, rec.InternalID); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.CompleteInterviewFailed)
	}
	if s.enqueue != nil {
		// 入队评估任务
		_ = s.enqueue.EnqueueInterviewEvaluate(ctx, sid)
	}

	// 更新会话缓存
	if s.cache != nil {
		sess, gerr := s.sessions.GetSessionBySessionID(ctx, sid)
		if gerr == nil && sess != nil {
			qjs, _ := json.Marshal(sess.Questions)
			adv := len(sess.Questions)
			if sess.TotalQuestions > 0 {
				adv = sess.TotalQuestions
			}
			rid := sess.ResumeID
			statusStr := string(sess.Status)
			// 更新会话缓存
			_ = s.cache.SaveSession(ctx, sess.SessionID, sess.ResumeText, &rid, string(qjs), sess.CurrentQuestionIndex, statusStr, &adv)
		}
	}
	return nil, nil
}
