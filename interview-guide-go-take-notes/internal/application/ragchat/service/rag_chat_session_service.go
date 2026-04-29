package service

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	kbresults "interview-guide-go/internal/application/knowledgebase/model/results"
	ragresults "interview-guide-go/internal/application/ragchat/model/results"
	ragrepo "interview-guide-go/internal/application/ragchat/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"gorm.io/gorm"
)

// RagChatSessionService RAG 会话 CRUD（与流式发消息解耦；stream 另实现）。
type RagChatSessionService struct {
	repo ragrepo.RagChatRepository
}

// NewRagChatSessionService Wire 注入
func NewRagChatSessionService(r ragrepo.RagChatRepository) *RagChatSessionService {
	return &RagChatSessionService{repo: r}
}

const (
	defaultRagSessionTitle = "新对话"
	maxRagSessionTitle     = 200
)

// Create POST /api/rag-chat/sessions
func (s *RagChatSessionService) Create(ctx context.Context, title string, knowledgeBaseIds []int64) (*ragresults.RagChatSession, error) {
	if s == nil || s.repo == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	ids := dedupeSorted(knowledgeBaseIds)
	if err := s.repo.ValidateKnowledgeBaseIDsExist(ctx, ids); err != nil {
		if errors.Is(err, ragrepo.ErrInvalidKnowledgeBaseIDs) {
			return nil, response.Err(http.StatusBadRequest, errmsg.RagChatInvalidKnowledgeBase)
		}
		return nil, err
	}
	t := strings.TrimSpace(title)
	if t == "" {
		t = defaultRagSessionTitle
	}
	if len([]rune(t)) > maxRagSessionTitle {
		return nil, response.Err(http.StatusBadRequest, errmsg.RagChatTitleTooLong)
	}
	sid, err := s.repo.CreateSessionWithKnowledgeBases(ctx, t, ids)
	if err != nil {
		return nil, err
	}
	srow, err := s.repo.GetSessionByID(ctx, sid)
	if err != nil {
		return nil, err
	}
	if srow == nil {
		return nil, response.Err(http.StatusInternalServerError, "rag chat session not found after create")
	}
	return &ragresults.RagChatSession{
		ID:               sid,
		Title:            t,
		KnowledgeBaseIds: ids,
		CreatedAt:        srow.CreatedAt,
	}, nil
}

// List GET /api/rag-chat/sessions
func (s *RagChatSessionService) List(ctx context.Context) ([]ragresults.RagChatSessionListItem, error) {
	if s == nil || s.repo == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	rows, err := s.repo.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ragresults.RagChatSessionListItem, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		ut := r.UpdatedAt
		if ut == nil {
			tm := r.CreatedAt
			ut = &tm
		}
		names := r.KnowledgeBaseNames
		if names == nil {
			names = []string{}
		}
		out = append(out, ragresults.RagChatSessionListItem{
			ID:                 r.ID,
			Title:              r.Title,
			MessageCount:       r.MessageCount,
			KnowledgeBaseNames: names,
			UpdatedAt:          ut.Format(time.RFC3339),
			IsPinned:           r.IsPinned,
		})
	}
	return out, nil
}

// GetDetail GET /api/rag-chat/sessions/{id}
func (s *RagChatSessionService) GetDetail(ctx context.Context, id int64) (*ragresults.RagChatSessionDetail, error) {
	if s == nil || s.repo == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if id < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid session id")
	}
	srow, err := s.repo.GetSessionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if srow == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
	}
	kbs, err := s.repo.ListKnowledgeBaseItemsForSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if kbs == nil {
		kbs = []kbresults.KnowledgeBaseListItem{}
	}
	msgRows, err := s.repo.ListMessagesBySessionID(ctx, id)
	if err != nil {
		return nil, err
	}
	msgOut := make([]ragresults.RagChatMessage, 0, len(msgRows))
	for i := range msgRows {
		msgOut = append(msgOut, ragresults.RagChatMessage{
			ID:        msgRows[i].ID,
			Type:      messageTypeForAPI(msgRows[i].Type),
			Content:   msgRows[i].Content,
			CreatedAt: msgRows[i].CreatedAt,
		})
	}
	return &ragresults.RagChatSessionDetail{
		ID:             srow.ID,
		Title:          srow.Title,
		KnowledgeBases: kbs,
		Messages:       msgOut,
		CreatedAt:      srow.CreatedAt,
		UpdatedAt:      srow.UpdatedAt,
	}, nil
}

// UpdateTitle PUT /api/rag-chat/sessions/{id}/title
func (s *RagChatSessionService) UpdateTitle(ctx context.Context, id int64, title string) error {
	if s == nil || s.repo == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if id < 1 {
		return response.Err(http.StatusBadRequest, "invalid session id")
	}
	t := strings.TrimSpace(title)
	if t == "" {
		return response.Err(http.StatusBadRequest, errmsg.RagChatTitleEmpty)
	}
	if len([]rune(t)) > maxRagSessionTitle {
		return response.Err(http.StatusBadRequest, errmsg.RagChatTitleTooLong)
	}
	if err := s.repo.UpdateSessionTitle(ctx, id, t); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
		}
		return err
	}
	return nil
}

// UpdateKnowledgeBases PUT /api/rag-chat/sessions/{id}/knowledge-bases
func (s *RagChatSessionService) UpdateKnowledgeBases(ctx context.Context, id int64, knowledgeBaseIds []int64) error {
	if s == nil || s.repo == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if id < 1 {
		return response.Err(http.StatusBadRequest, "invalid session id")
	}
	ids := dedupeSorted(knowledgeBaseIds)
	if err := s.repo.ValidateKnowledgeBaseIDsExist(ctx, ids); err != nil {
		if errors.Is(err, ragrepo.ErrInvalidKnowledgeBaseIDs) {
			return response.Err(http.StatusBadRequest, errmsg.RagChatInvalidKnowledgeBase)
		}
		return err
	}
	srow, err := s.repo.GetSessionByID(ctx, id)
	if err != nil {
		return err
	}
	if srow == nil {
		return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
	}
	return s.repo.ReplaceSessionKnowledgeBases(ctx, id, ids)
}

// TogglePin PUT /api/rag-chat/sessions/{id}/pin
func (s *RagChatSessionService) TogglePin(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if id < 1 {
		return response.Err(http.StatusBadRequest, "invalid session id")
	}
	if srow, err := s.repo.GetSessionByID(ctx, id); err != nil {
		return err
	} else if srow == nil {
		return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
	}
	if err := s.repo.ToggleSessionPin(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
		}
		return err
	}
	return nil
}

// Delete DELETE /api/rag-chat/sessions/{id}
func (s *RagChatSessionService) Delete(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if id < 1 {
		return response.Err(http.StatusBadRequest, "invalid session id")
	}
	if err := s.repo.DeleteSession(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Err(http.StatusNotFound, errmsg.RagChatSessionNotFound)
		}
		return err
	}
	return nil
}

func dedupeSorted(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	uniq := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i] < uniq[j] })
	return uniq
}

func messageTypeForAPI(t string) string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "USER":
		return "user"
	case "ASSISTANT":
		return "assistant"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}
