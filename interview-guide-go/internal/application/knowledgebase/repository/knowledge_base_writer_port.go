package repository

import (
	"context"
	"errors"

	resumerepo "interview-guide-go/internal/application/resume/repository"
)

// ErrKnowledgeBaseDeleteNoRow 删除时未影响行（如并发下记录已不存在）。
var ErrKnowledgeBaseDeleteNoRow = errors.New("knowledge base delete: no row affected")

// ErrKnowledgeBaseUpdateNoRow 更新分类时未影响行（如并发下记录已不存在）。
var ErrKnowledgeBaseUpdateNoRow = errors.New("knowledge base update: no row affected")

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

// KnowledgeBaseChunkInsert 单条分块向量（消费者写入；ChunkIndex 从 0 连续）。
type KnowledgeBaseChunkInsert struct {
	ChunkIndex int
	Content    string
	Embedding  []float32
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
	// 文件哈希命中已有记录时的去重处理：access_count+1。
	IncrementAccessCount(ctx context.Context, id int64) error
	// 	入队失败或消费者回写时用。
	UpdateVectorStatus(ctx context.Context, id int64, status, errMsg string) error
	// GetVectorMetaByID 供向量化消费者判断记录是否存在及当前 vector_status。
	GetVectorMetaByID(ctx context.Context, id int64) (vectorStatus string, found bool, err error)
	// MarkVectorizationComplete 消费者分块成功后回写 COMPLETED、chunk_count，并清空 vector_error。
	MarkVectorizationComplete(ctx context.Context, id int64, chunkCount int) error
	// SaveKnowledgeBaseVectorChunks 事务内删除该 KB 既有分块、写入新向量并置 COMPLETED/chunk_count（与 embedding 写入原子一致）。
	SaveKnowledgeBaseVectorChunks(ctx context.Context, knowledgeBaseID int64, chunks []KnowledgeBaseChunkInsert) error
	// DeleteKnowledgeBaseByID 删除知识库行；`rag_session_knowledge_bases` 上 ON DELETE CASCADE 会一并清关联。
	DeleteKnowledgeBaseByID(ctx context.Context, id int64) error
	// UpdateKnowledgeBaseCategory 更新 category 列；空串表示未分类。未影响行时返回 ErrKnowledgeBaseUpdateNoRow。
	UpdateKnowledgeBaseCategory(ctx context.Context, id int64, category string) error
	// IncrementQuestionCounts 将所列知识库的 question_count 各 +1，用于命中后的访问计数。
	IncrementQuestionCounts(ctx context.Context, ids []int64) error
}

// ObjectStoragePort 复用简历对象存储端口，上传/预签名与 bucket 约定一致。
type ObjectStoragePort = resumerepo.ObjectStoragePort
