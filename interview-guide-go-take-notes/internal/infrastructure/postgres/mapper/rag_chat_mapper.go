package mapper

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	kbresults "interview-guide-go/internal/application/knowledgebase/model/results"
	ragrepo "interview-guide-go/internal/application/ragchat/repository"
	grommodel "interview-guide-go/internal/infrastructure/postgres/grom"

	"gorm.io/gorm"
)

// RagChatMapper RAG 会话与关联表。
type RagChatMapper struct {
	db *gorm.DB
}

// NewRagChatMapper 由 Wire 注入。
func NewRagChatMapper(db *gorm.DB) *RagChatMapper {
	return &RagChatMapper{db: db}
}

var _ ragrepo.RagChatRepository = (*RagChatMapper)(nil)

// CreateSessionWithKnowledgeBases 插入会话与 rag_session_knowledge_bases。
func (m *RagChatMapper) CreateSessionWithKnowledgeBases(ctx context.Context, title string, kbIDs []int64) (int64, error) {
	if m == nil || m.db == nil {
		return 0, errors.New("db not configured")
	}
	var id int64
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sess := grommodel.RagChatSession{
			Title:        title,
			Status:       "ACTIVE",
			MessageCount: 0,
			IsPinned:     false,
		}
		if err := tx.Create(&sess).Error; err != nil {
			return err
		}
		id = sess.ID
		for _, kid := range kbIDs {
			row := grommodel.RagSessionKnowledgeBase{SessionID: id, KnowledgeBaseID: kid}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ValidateKnowledgeBaseIDsExist 要求 ids 去重后逐个存在。
func (m *RagChatMapper) ValidateKnowledgeBaseIDsExist(ctx context.Context, ids []int64) error {
	if m == nil || m.db == nil {
		return errors.New("db not configured")
	}
	if len(ids) == 0 {
		return nil
	}
	var n int64
	if err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).Where("id IN ?", ids).Count(&n).Error; err != nil {
		return err
	}
	if int64(len(ids)) != n {
		return ragrepo.ErrInvalidKnowledgeBaseIDs
	}
	return nil
}

// ListSessions 置顶优先，再按「最近活动」时间倒序。
func (m *RagChatMapper) ListSessions(ctx context.Context) ([]ragrepo.RagSessionListRow, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("db not configured")
	}
	var sessions []grommodel.RagChatSession
	err := m.db.WithContext(ctx).Model(&grommodel.RagChatSession{}).
		Order("is_pinned DESC").
		Order("COALESCE(updated_at, created_at) DESC").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return []ragrepo.RagSessionListRow{}, nil
	}
	sessIDs := make([]int64, len(sessions))
	for i := range sessions {
		sessIDs[i] = sessions[i].ID
	}
	type nameRow struct {
		SessionID int64  `gorm:"column:session_id"`
		Name      string `gorm:"column:name"`
	}
	var nrows []nameRow
	_ = m.db.WithContext(ctx).Table("rag_session_knowledge_bases as rsk").
		Select("rsk.session_id, kb.name as name").
		Joins("JOIN knowledge_bases kb ON kb.id = rsk.knowledge_base_id").
		Where("rsk.session_id IN ?", sessIDs).
		Scan(&nrows).Error
	bySess := make(map[int64][]string)
	for _, r := range nrows {
		if strings.TrimSpace(r.Name) == "" {
			continue
		}
		bySess[r.SessionID] = append(bySess[r.SessionID], r.Name)
	}
	for k, names := range bySess {
		sort.Strings(names)
		bySess[k] = names
	}
	out := make([]ragrepo.RagSessionListRow, 0, len(sessions))
	for i := range sessions {
		s := &sessions[i]
		names := bySess[s.ID]
		if names == nil {
			names = []string{}
		}
		out = append(out, ragrepo.RagSessionListRow{
			ID:                 s.ID,
			Title:              s.Title,
			MessageCount:       s.MessageCount,
			IsPinned:           s.IsPinned,
			CreatedAt:          s.CreatedAt,
			UpdatedAt:          s.UpdatedAt,
			KnowledgeBaseNames: names,
		})
	}
	return out, nil
}

// GetSessionByID 单条；不存在 (nil, nil)。
func (m *RagChatMapper) GetSessionByID(ctx context.Context, id int64) (*ragrepo.RagSessionRow, error) {
	if m == nil || m.db == nil || id < 1 {
		return nil, errors.New("invalid id")
	}
	var row grommodel.RagChatSession
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ragrepo.RagSessionRow{
		ID:           row.ID,
		Title:        row.Title,
		MessageCount: row.MessageCount,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

// ListMessagesBySessionID 按 message_order 升序。
func (m *RagChatMapper) ListMessagesBySessionID(ctx context.Context, sessionID int64) ([]ragrepo.RagMessageRow, error) {
	if m == nil || m.db == nil || sessionID < 1 {
		return nil, errors.New("invalid session id")
	}
	var rows []grommodel.RagChatMessage
	err := m.db.WithContext(ctx).Model(&grommodel.RagChatMessage{}).
		Where("session_id = ?", sessionID).
		Order("message_order ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ragrepo.RagMessageRow, 0, len(rows))
	for i := range rows {
		out = append(out, ragrepo.RagMessageRow{
			ID:        rows[i].ID,
			Type:      rows[i].Type,
			Content:   rows[i].Content,
			CreatedAt: rows[i].CreatedAt,
		})
	}
	return out, nil
}

// ListKnowledgeBaseItemsForSession 关联的 knowledge_bases 全字段列表项。
func (m *RagChatMapper) ListKnowledgeBaseItemsForSession(ctx context.Context, sessionID int64) ([]kbresults.KnowledgeBaseListItem, error) {
	if m == nil || m.db == nil || sessionID < 1 {
		return nil, errors.New("invalid session id")
	}
	var krows []grommodel.KnowledgeBase
	sub := m.db.Model(&grommodel.RagSessionKnowledgeBase{}).
		Select("knowledge_base_id").
		Where("session_id = ?", sessionID)
	err := m.db.WithContext(ctx).Model(&grommodel.KnowledgeBase{}).
		Where("id IN (?)", sub).
		Order("id ASC").
		Find(&krows).Error
	if err != nil {
		return nil, err
	}
	out := make([]kbresults.KnowledgeBaseListItem, 0, len(krows))
	for i := range krows {
		out = append(out, kbRowToListItem(&krows[i]))
	}
	return out, nil
}

// ReplaceSessionKnowledgeBases 替换会话绑定的知识库 id 集合（先全删后插）。
func (m *RagChatMapper) ReplaceSessionKnowledgeBases(ctx context.Context, sessionID int64, kbIDs []int64) error {
	if m == nil || m.db == nil || sessionID < 1 {
		return errors.New("invalid session id")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", sessionID).Delete(&grommodel.RagSessionKnowledgeBase{}).Error; err != nil {
			return err
		}
		for _, kid := range kbIDs {
			if err := tx.Create(&grommodel.RagSessionKnowledgeBase{SessionID: sessionID, KnowledgeBaseID: kid}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&grommodel.RagChatSession{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}
		return nil
	})
}

// UpdateSessionTitle 仅更新 title；GORM BeforeUpdate 会写 updated_at。
func (m *RagChatMapper) UpdateSessionTitle(ctx context.Context, sessionID int64, title string) error {
	if m == nil || m.db == nil || sessionID < 1 {
		return errors.New("invalid session id")
	}
	res := m.db.WithContext(ctx).Model(&grommodel.RagChatSession{}).Where("id = ?", sessionID).Updates(map[string]interface{}{
		"title":      title,
		"updated_at": time.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ToggleSessionPin 翻转 is_pinned。
func (m *RagChatMapper) ToggleSessionPin(ctx context.Context, sessionID int64) error {
	if m == nil || m.db == nil || sessionID < 1 {
		return errors.New("invalid session id")
	}
	res := m.db.WithContext(ctx).Exec(`
		UPDATE rag_chat_sessions
		SET is_pinned = NOT is_pinned, updated_at = NOW()
		WHERE id = ?`, sessionID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteSession 级联删消息与关联（FK ON DELETE CASCADE）。
func (m *RagChatMapper) DeleteSession(ctx context.Context, sessionID int64) error {
	if m == nil || m.db == nil || sessionID < 1 {
		return errors.New("invalid session id")
	}
	res := m.db.WithContext(ctx).Delete(&grommodel.RagChatSession{}, sessionID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func kbRowToListItem(row *grommodel.KnowledgeBase) kbresults.KnowledgeBaseListItem {
	var cat *string
	if t := strings.TrimSpace(row.Category); t != "" {
		cat = &t
	}
	fs := int64(0)
	if row.FileSize != nil {
		fs = *row.FileSize
	}
	ve := strings.TrimSpace(row.VectorError)
	return kbresults.KnowledgeBaseListItem{
		ID:               row.ID,
		Name:             row.Name,
		Category:         cat,
		OriginalFilename: row.OriginalFilename,
		FileSize:         fs,
		ContentType:      strings.TrimSpace(row.ContentType),
		UploadedAt:       row.UploadedAt,
		LastAccessedAt:   row.LastAccessedAt,
		AccessCount:      row.AccessCount,
		QuestionCount:    row.QuestionCount,
		VectorStatus:     strings.TrimSpace(row.VectorStatus),
		VectorError:      ve,
		ChunkCount:       row.ChunkCount,
	}
}
