package model

import (
	"time"

	"gorm.io/gorm"
)

// KnowledgeBase 对应 KnowledgeBaseEntity，表 knowledge_bases。
type KnowledgeBase struct {
	ID               int64      `gorm:"column:id;primaryKey;autoIncrement"`
	FileHash         string     `gorm:"column:file_hash;type:varchar(64);uniqueIndex:idx_kb_hash;not null"`
	Name             string     `gorm:"column:name;not null"`
	Category         string     `gorm:"column:category;size:100;index:idx_kb_category"`
	OriginalFilename string     `gorm:"column:original_filename;not null"`
	FileSize         *int64     `gorm:"column:file_size"`
	ContentType      string     `gorm:"column:content_type"`
	StorageKey       string     `gorm:"column:storage_key;size:500"`
	StorageURL       string     `gorm:"column:storage_url;size:1000"`
	UploadedAt       time.Time  `gorm:"column:uploaded_at"`
	LastAccessedAt   *time.Time `gorm:"column:last_accessed_at"`
	AccessCount      int        `gorm:"column:access_count;default:1"`
	QuestionCount    int        `gorm:"column:question_count;default:0"`
	VectorStatus     string     `gorm:"column:vector_status;size:20;default:PENDING"`
	VectorError      string     `gorm:"column:vector_error;size:500"`
	ChunkCount       int        `gorm:"column:chunk_count;default:0"`
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

func (k *KnowledgeBase) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if k.UploadedAt.IsZero() {
		k.UploadedAt = now
	}
	if k.LastAccessedAt == nil {
		k.LastAccessedAt = &now
	}
	if k.AccessCount == 0 {
		k.AccessCount = 1
	}
	return nil
}
