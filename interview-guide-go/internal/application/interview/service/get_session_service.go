package service

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// GetSessionService GET /sessions/{sessionId}：按对外 sessionId 返回当前会话快照（题目、游标、状态，含已答合并）。
type GetSessionService struct {
	sessions repository.InterviewSessionWriter
}

// NewGetSessionService 由 wire 注入。
func NewGetSessionService(sessions repository.InterviewSessionWriter) *GetSessionService {
	return &GetSessionService{sessions: sessions}
}

// GetSession 与主项目 GetSession 语义一致：加载会话、合并 interview_answers，供详情/轮询使用。
func (s *GetSessionService) GetSession(ctx context.Context, sessionID string) (*results.InterviewSession, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview session not configured")
	}

	// 获取会话
	out, err := s.sessions.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetInterviewSessionFailed)
	}
	if out == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SessionNotFound)
	}
	return out, nil
}
