package service

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// UnfinishedSessionService 只读：按 resume 查未终态会话（恢复续答等）。
type UnfinishedSessionService struct {
	sessions repository.InterviewSessionWriter
}

func NewUnfinishedSessionService(sessions repository.InterviewSessionWriter) *UnfinishedSessionService {
	return &UnfinishedSessionService{sessions: sessions}
}

// FindUnfinishedSession 按 resumeId 查最近一条未终态会话；无则 404。resumeID 须已由 controller 校验 >=1。
func (s *UnfinishedSessionService) FindUnfinishedSession(ctx context.Context, resumeID int64) (*results.InterviewSession, error) {
	sess, err := s.sessions.FindUnfinishedSession(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.FindUnfinishedNotFound)
	}
	return sess, nil
}
