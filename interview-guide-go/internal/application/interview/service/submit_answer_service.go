package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/interview"
	"interview-guide-go/shared/response"
)

// SubmitAnswerService 处理 POST /sessions/{id}/answers（与主项目 SubmitAnswer 对齐）。
type SubmitAnswerService struct {
	sessions repository.InterviewSessionWriter
	cache    repository.InterviewSessionCache
	enqueue  repository.InterviewEvaluateEnqueuer
}

func NewSubmitAnswerService(
	sessions repository.InterviewSessionWriter,
	cache repository.InterviewSessionCache,
	enqueue repository.InterviewEvaluateEnqueuer,
) *SubmitAnswerService {
	return &SubmitAnswerService{sessions: sessions, cache: cache, enqueue: enqueue}
}

// SubmitAnswer POST 提交单题答案并推进游标。in 须由 controller 完成入参校验与 Trim；题号上界在加载题目后校验。
func (s *SubmitAnswerService) SubmitAnswer(ctx context.Context, in model.ValidatedSubmitAnswer) (*results.SubmitAnswerResponse, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview writer not configured")
	}

	rec, err := s.sessions.GetSessionRecordForSubmit(ctx, in.SessionID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}
	st := strings.ToUpper(strings.TrimSpace(rec.Status))
	if st == "QUESTIONS_PENDING" || st == "QUESTIONS_FAILED" || st == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}
	if st == string(interview.StatusCompleted) || st == string(interview.StatusEvaluated) {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerSessionClosed)
	}

	raw := strings.TrimSpace(rec.QuestionsJSON)
	if raw == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}

	// 解析面试题目
	var qs []results.InterviewQuestion
	if err := json.Unmarshal([]byte(raw), &qs); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
	}
	if len(qs) == 0 {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}
	if in.QuestionIndex >= len(qs) {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerBadQuestionIndex)
	}

	q := qs[in.QuestionIndex]

	// 默认评分 0 分
	score0 := 0
	if err := s.sessions.SaveInterviewAnswer(ctx, rec.InternalID, in.QuestionIndex, q.Question, q.Category, in.Answer, &score0, ""); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
	}
	ua := in.Answer
	qs[in.QuestionIndex].UserAnswer = &ua

	newIdx := in.QuestionIndex + 1
	hasNext := newIdx < len(qs)
	newStatus := string(interview.StatusInProgress)
	setCompleted := false
	if !hasNext {
		newStatus = string(interview.StatusCompleted)
		setCompleted = true

		// 更新评估状态为 PENDING
		if err := s.sessions.UpdateInterviewSessionEvaluatePending(ctx, rec.InternalID); err != nil {
			return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
		}

		// 入队评估任务
		if s.enqueue != nil {
			_ = s.enqueue.EnqueueInterviewEvaluate(ctx, in.SessionID)
		}
	}
	if err := s.sessions.UpdateInterviewSessionProgress(ctx, rec.InternalID, newIdx, newStatus, setCompleted); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
	}

	// 更新缓存
	if s.cache != nil {
		adv := len(qs)
		if rec.TotalQuestions != nil && *rec.TotalQuestions > 0 {
			adv = *rec.TotalQuestions
		}
		if qjs, mErr := json.Marshal(qs); mErr == nil {
			_ = s.cache.SaveSession(ctx, rec.SessionID, rec.ResumeText, &rec.ResumeID, string(qjs), newIdx, newStatus, &adv)
		}
	}

	// 返回下一题
	var next *results.InterviewQuestion
	if hasNext {
		nq := qs[newIdx]
		next = &nq
	}
	return &results.SubmitAnswerResponse{
		HasNextQuestion: hasNext,
		NextQuestion:    next,
		CurrentIndex:    newIdx,
		TotalQuestions:  len(qs),
	}, nil
}

// SaveAnswer PUT 仅将本题答案落库为草稿/覆盖：不推进游标、不改变会话完成态、不触发评估入队；允许 answer 空串以清空已保存的草稿行。
func (s *SubmitAnswerService) SaveAnswer(ctx context.Context, in model.ValidatedSaveAnswer) (*results.SubmitAnswerResponse, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview writer not configured")
	}
	rec, err := s.sessions.GetSessionRecordForSubmit(ctx, in.SessionID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}
	st := strings.ToUpper(strings.TrimSpace(rec.Status))
	if st == "QUESTIONS_PENDING" || st == "QUESTIONS_FAILED" || st == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}
	if st == string(interview.StatusCompleted) || st == string(interview.StatusEvaluated) {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerSessionClosed)
	}
	raw := strings.TrimSpace(rec.QuestionsJSON)
	if raw == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}
	var qs []results.InterviewQuestion
	if err := json.Unmarshal([]byte(raw), &qs); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
	}
	if len(qs) == 0 {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerQuestionsNotReady)
	}
	if in.QuestionIndex < 0 || in.QuestionIndex >= len(qs) {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerBadQuestionIndex)
	}
	q := qs[in.QuestionIndex]
	score0 := 0
	if err := s.sessions.SaveInterviewAnswer(ctx, rec.InternalID, in.QuestionIndex, q.Question, q.Category, in.Answer, &score0, ""); err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.SubmitAnswerPersistFailed)
	}
	// 合并回内存用于刷新 Redis 缓存，游标/状态与 DB 保持一致
	ua := in.Answer
	qs[in.QuestionIndex].UserAnswer = &ua
	curIdx := rec.CurrentQuestionIndex
	sessionStatus := strings.TrimSpace(rec.Status)
	adv := len(qs)
	if rec.TotalQuestions != nil && *rec.TotalQuestions > 0 {
		adv = *rec.TotalQuestions
	}
	if s.cache != nil {
		if qjs, mErr := json.Marshal(qs); mErr == nil {
			_ = s.cache.SaveSession(ctx, rec.SessionID, rec.ResumeText, &rec.ResumeID, string(qjs), curIdx, sessionStatus, &adv)
		}
	}
	hasNext := in.QuestionIndex+1 < len(qs)
	var next *results.InterviewQuestion
	if hasNext {
		nq := qs[in.QuestionIndex+1]
		next = &nq
	}
	return &results.SubmitAnswerResponse{
		HasNextQuestion: hasNext,
		NextQuestion:    next,
		CurrentIndex:    curIdx,
		TotalQuestions:  len(qs),
	}, nil
}
