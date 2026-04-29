package service

import (
	"context"
	"errors"
	"net/http"

	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"gorm.io/gorm"
)

// DeleteInterviewService 删除一场面试会话及其 Redis 侧键与答题子表，确保级联清理。
type DeleteInterviewService struct {
	sessions repository.InterviewSessionWriter
	cache    repository.InterviewSessionCache
}

// NewDeleteInterviewService cache 可为 nil（未接 Redis 时仅删库）。
func NewDeleteInterviewService(sessions repository.InterviewSessionWriter, cache repository.InterviewSessionCache) *DeleteInterviewService {
	return &DeleteInterviewService{sessions: sessions, cache: cache}
}

// DeleteSession 按对外 sessionId 删除会话；成功返回 { "message": "..." }，便于前端弹提示。
func (s *DeleteInterviewService) DeleteSession(ctx context.Context, sid string) (map[string]string, error) {
	// 1. 获取会话
	sess, err := s.sessions.GetSessionBySessionID(ctx, sid)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.DeleteInterviewSessionNotFound)
	}

	// 2. 删除缓存
	if s.cache != nil {
		if cErr := s.cache.DeleteSessionKeys(ctx, sess.SessionID, sess.ResumeID); cErr != nil {
			return nil, cErr
		}
	}

	// 3. 删除会话
	if err := s.sessions.DeleteInterviewSessionByPublicID(ctx, sid); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.Err(http.StatusNotFound, errmsg.DeleteInterviewSessionNotFound)
		}
		return nil, response.Err(http.StatusInternalServerError, errmsg.DeleteInterviewSessionFailed)
	}
	return map[string]string{"message": errmsg.DeleteInterviewSessionMessage}, nil
}
