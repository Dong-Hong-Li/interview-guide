package repository

import (
	"context"
	"errors"
	"time"

	kbresults "interview-guide-go/internal/application/knowledgebase/model/results"
)

// ErrInvalidKnowledgeBaseIDs 入参中的知识库 id 在库中未全部存在。
var ErrInvalidKnowledgeBaseIDs = errors.New("invalid knowledge base ids")

// RagChatRepository RAG 会话与消息、会话-知识库关联（由 postgres/mapper 实现）。
type RagChatRepository interface {
	// CreateSessionWithKnowledgeBases 插入会话并写入关联行；title 已 trim；kbIDs 已去重。
	CreateSessionWithKnowledgeBases(ctx context.Context, title string, kbIDs []int64) (sessionID int64, err error)
	// ValidateKnowledgeBaseIDsExist 所有 id 均存在时返回 nil，否则返回业务错误由 service 映射。
	ValidateKnowledgeBaseIDsExist(ctx context.Context, ids []int64) error
	// ListSessions 置顶优先，再按最近活动时间倒序。
	ListSessions(ctx context.Context) ([]RagSessionListRow, error)
	// GetSessionByID 不存在返回 (nil, nil)。
	GetSessionByID(ctx context.Context, id int64) (*RagSessionRow, error)
	// ListMessagesBySessionID 按 message_order 升序。
	ListMessagesBySessionID(ctx context.Context, sessionID int64) ([]RagMessageRow, error)
	// ListKnowledgeBaseItemsForSession 详情区「已选知识库」完整元数据。
	ListKnowledgeBaseItemsForSession(ctx context.Context, sessionID int64) ([]kbresults.KnowledgeBaseListItem, error)
	// ListKnowledgeBaseIDsForSession 会话绑定的知识库主键（升序）；无绑定返回空切片。
	ListKnowledgeBaseIDsForSession(ctx context.Context, sessionID int64) ([]int64, error)
	// InsertChatMessage 追加一条消息并 session.message_count+1、updated_at 刷新；typ 为 USER / ASSISTANT。
	InsertChatMessage(ctx context.Context, sessionID int64, typ string, content string) error
	// ReplaceSessionKnowledgeBases 先删后插。
	ReplaceSessionKnowledgeBases(ctx context.Context, sessionID int64, kbIDs []int64) error
	UpdateSessionTitle(ctx context.Context, sessionID int64, title string) error
	ToggleSessionPin(ctx context.Context, sessionID int64) error
	DeleteSession(ctx context.Context, sessionID int64) error
}

// RagSessionListRow 列表行（含聚合出的知识库名称，供 JSON 与前端对齐）。
type RagSessionListRow struct {
	ID                 int64
	Title              string
	MessageCount       int
	IsPinned           bool
	CreatedAt          time.Time
	UpdatedAt          *time.Time
	KnowledgeBaseNames []string
}

// RagSessionRow 单条会话（无关联名）。
type RagSessionRow struct {
	ID           int64
	Title        string
	MessageCount int
	CreatedAt    time.Time
	UpdatedAt    *time.Time
}

// RagMessageRow 消息行（type 存 user/assistant，与前端一致）。
type RagMessageRow struct {
	ID        int64
	Type      string
	Content   string
	CreatedAt time.Time
}
