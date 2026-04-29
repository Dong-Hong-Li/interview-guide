package repository

import "context"

// VectorizeTaskPublisher 向 Redis Stream 投递知识库向量化任务（kbId + 全文 content）。
type VectorizeTaskPublisher interface {
	SendVectorizeTask(ctx context.Context, kbID int64, content string) error
}
