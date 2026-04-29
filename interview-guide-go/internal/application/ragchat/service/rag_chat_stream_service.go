package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	kbmodel "interview-guide-go/internal/application/knowledgebase/model"
	kbsvc "interview-guide-go/internal/application/knowledgebase/service"
	ragrepo "interview-guide-go/internal/application/ragchat/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/response"

	"go.uber.org/zap"
)

// RagChatStreamService POST /api/rag-chat/sessions/{sessionId}/messages/stream：
// 读取会话绑定知识库 → 写入 USER 消息 → 复用 KnowledgeBaseQueryService 做向量检索 + SSE → 写入 ASSISTANT 消息。
type RagChatStreamService struct {
	lg      *zap.Logger
	repo    ragrepo.RagChatRepository
	kbQuery *kbsvc.KnowledgeBaseQueryService
}

// NewRagChatStreamService Wire 注入。
func NewRagChatStreamService(lg *zap.Logger, repo ragrepo.RagChatRepository, kbQuery *kbsvc.KnowledgeBaseQueryService) *RagChatStreamService {
	return &RagChatStreamService{lg: lg, repo: repo, kbQuery: kbQuery}
}

// StreamSessionMessage 输出 text/event-stream；成功后在库中追加 USER（先发）与 ASSISTANT（补全后）。
func (s *RagChatStreamService) StreamSessionMessage(ctx context.Context, sessionID int64, question string, w io.Writer, flush func()) error {
	if s == nil || s.repo == nil || s.kbQuery == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.RagChatStreamServiceNil)
	}
	q := strings.TrimSpace(question)
	if q == "" {
		return response.Err(http.StatusBadRequest, errmsg.RagChatQuestionEmpty)
	}
	if sessionID < 1 {
		return response.Err(http.StatusBadRequest, errmsg.RagChatInvalidSessionPathID)
	}

	row, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if row == nil {
		return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
	}

	kbIDs, err := s.repo.ListKnowledgeBaseIDsForSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(kbIDs) == 0 {
		return response.Err(http.StatusBadRequest, errmsg.RagChatSessionNoKnowledgeBases)
	}

	if err := s.repo.InsertChatMessage(ctx, sessionID, "USER", q); err != nil {
		return err
	}

	var acc strings.Builder
	v := &kbmodel.ValidatedKBQuery{KnowledgeBaseIDs: kbIDs, Question: q}
	err = s.kbQuery.QueryStream(ctx, v, w, flush, &acc)
	if err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgRagChatStreamFailed,
				zap.Int64("sessionId", sessionID),
				zap.Error(err),
			)
		}
		return err
	}

	ans := strings.TrimSpace(acc.String())
	if err := s.repo.InsertChatMessage(ctx, sessionID, "ASSISTANT", ans); err != nil {
		if s.lg != nil {
			s.lg.Warn(logmsg.MsgRagChatPersistAssistantFailed, zap.Int64("sessionId", sessionID), zap.Error(err))
		}
		return err
	}
	if s.lg != nil {
		s.lg.Info(logmsg.MsgRagChatStreamOK,
			zap.Int64("sessionId", sessionID),
			zap.Any("knowledgeBaseIds", kbIDs),
			zap.Int("questionRunes", utf8.RuneCountInString(q)),
			zap.Int("answerRunes", utf8.RuneCountInString(ans)),
		)
	}
	return nil
}
