package repository

import "context"

// KnowledgeTextEmbedder 知识库分块向量（由基础设施注入 Redis Stream 消费者）。
type KnowledgeTextEmbedder interface {
	// Embed 与 texts 同序返回向量；任一输入失败返回错误。
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
