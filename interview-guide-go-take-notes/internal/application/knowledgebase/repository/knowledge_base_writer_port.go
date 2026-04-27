package repository

import (
	"context"

	resumerepo "interview-guide-go/internal/application/resume/repository"
)

// KnowledgeBaseInsert 写入 knowledge_bases 一行（与 ORM 解耦）。
type KnowledgeBaseInsert struct {
	FileHash         string
	Name             string
	Category         string
	OriginalFilename string
	FileSize         int64
	ContentType      string
	StorageKey       string
	StorageURL       string
	VectorStatus     string
}

// ExistingKnowledgeBase 去重命中时返回的已有行子集。
type ExistingKnowledgeBase struct {
	ID           int64
	Name         string
	Category     string
	FileSize     int64
	StorageKey   string
	StorageURL   string
	VectorStatus string
}

// KnowledgeBaseWriter 知识库持久化端口（由 postgres/mapper 实现）。
type KnowledgeBaseWriter interface {
	// 按 file_hash 唯一键查重；未命中返回 (nil, nil)。
	FindByFileHash(ctx context.Context, fileHash string) (*ExistingKnowledgeBase, error)
	// 插入一行并返回主键。
	InsertKnowledgeBase(ctx context.Context, in *KnowledgeBaseInsert) (id int64, err error)
	// 重复上传时与 Java handleDuplicateKnowledgeBase 一致：access_count+1。
	IncrementAccessCount(ctx context.Context, id int64) error
	// 	入队失败或消费者回写时用。
	UpdateVectorStatus(ctx context.Context, id int64, status, errMsg string) error
	// GetVectorMetaByID 供向量化消费者判断记录是否存在及当前 vector_status。
	GetVectorMetaByID(ctx context.Context, id int64) (vectorStatus string, found bool, err error)
	// MarkVectorizationComplete 消费者分块成功后回写 COMPLETED、chunk_count，并清空 vector_error。
	MarkVectorizationComplete(ctx context.Context, id int64, chunkCount int) error
}

// ObjectStoragePort 复用简历对象存储端口，上传/预签名与 bucket 约定一致。
type ObjectStoragePort = resumerepo.ObjectStoragePort
