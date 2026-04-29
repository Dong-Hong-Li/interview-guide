package model

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

// KnowledgeBaseChunk 对应表 knowledge_base_chunks；向量维度须与迁移 vector(N) 一致。
type KnowledgeBaseChunk struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement"`
	KnowledgeBaseID int64           `gorm:"column:knowledge_base_id;not null;index:idx_kb_chunks_kb_id"`
	ChunkIndex      int             `gorm:"column:chunk_index;not null"`
	Content         string          `gorm:"column:content;type:text;not null"`
	Embedding       pgvector.Vector `gorm:"column:embedding;type:vector(1536);not null"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (KnowledgeBaseChunk) TableName() string {
	return "knowledge_base_chunks"
}
