package service

import (
	"context"
	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/interview"
	"interview-guide-go/shared/response"
	"net/http"
	"strings"
)

// CurrentQuestionService 只读：按 sessionId 取当前应答题 / 轮询题是否就绪（与主项目 GetCurrentQuestion 对齐）。
type CurrentQuestionService struct {
	sessions repository.InterviewSessionWriter
}

func NewCurrentQuestionService(sessions repository.InterviewSessionWriter) *CurrentQuestionService {
	return &CurrentQuestionService{sessions: sessions}
}

// GetCurrentQuestion 返回当前应答题。sessionID 须已由 controller 校验非空并 Trim。
func (s *CurrentQuestionService) GetCurrentQuestion(ctx context.Context, sessionID string) (*results.CurrentQuestionResponse, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview session not configured")
	}
	sess, err := s.sessions.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetInterviewSessionFailed)
	}
	if sess == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SessionNotFound)
	}
	st := strings.TrimSpace(string(sess.Status))
	if len(sess.Questions) == 0 {
		out := &results.CurrentQuestionResponse{Completed: false}
		switch st {
		case "QUESTIONS_PENDING":
			out.Message = errmsg.QuestionsPendingMessage
		case "QUESTIONS_FAILED":
			out.Message = errmsg.QuestionsFailedMessage
		default:
			out.Message = errmsg.QuestionsNotReady
		}
		return out, nil
	}
	if st == string(interview.StatusCompleted) || st == string(interview.StatusEvaluated) {
		return &results.CurrentQuestionResponse{Completed: true, Message: errmsg.GetCurrentQuestionNoMore}, nil
	}
	idx := sess.CurrentQuestionIndex
	if idx < 0 || idx >= len(sess.Questions) {
		return &results.CurrentQuestionResponse{Completed: true, Message: errmsg.GetCurrentQuestionNoMore}, nil
	}
	q := sess.Questions[idx]
	return &results.CurrentQuestionResponse{Completed: false, Question: &q}, nil
}
