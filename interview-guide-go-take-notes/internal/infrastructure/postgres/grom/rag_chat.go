package model

import (
	"time"

	"gorm.io/gorm"
)

// RagChatSession 对应 RagChatSessionEntity，表 rag_chat_sessions。
type RagChatSession struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Title        string     `gorm:"column:title;not null"`
	Status       string     `gorm:"column:status;size:20;default:ACTIVE"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at;index:idx_rag_session_updated"`
	MessageCount int        `gorm:"column:message_count;default:0"`
	IsPinned     bool       `gorm:"column:is_pinned;default:false"`
}

func (RagChatSession) TableName() string { return "rag_chat_sessions" }

func (s *RagChatSession) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt == nil {
		t := now
		s.UpdatedAt = &t
	}
	return nil
}

func (s *RagChatSession) BeforeUpdate(_ *gorm.DB) error {
	now := time.Now()
	s.UpdatedAt = &now
	return nil
}

// RagChatMessage 对应 RagChatMessageEntity，表 rag_chat_messages。
type RagChatMessage struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
	SessionID    int64      `gorm:"column:session_id;not null;index:idx_rag_message_session;index:idx_rag_message_order,priority:1"`
	Type         string     `gorm:"column:type;size:20;not null"`
	Content      string     `gorm:"column:content;type:text;not null"`
	MessageOrder int        `gorm:"column:message_order;not null;index:idx_rag_message_order,priority:2"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    *time.Time `gorm:"column:updated_at"`
	Completed    bool       `gorm:"column:completed;default:true"`
}

func (RagChatMessage) TableName() string { return "rag_chat_messages" }

func (m *RagChatMessage) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt == nil {
		t := now
		m.UpdatedAt = &t
	}
	return nil
}
