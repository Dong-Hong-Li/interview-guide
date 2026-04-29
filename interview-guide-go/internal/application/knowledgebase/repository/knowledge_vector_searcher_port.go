package repository

import (
	"context"
)

// KnowledgeChunkHit 单次向量检索命中的一条分块（供 RAG 拼装 context）。
type KnowledgeChunkHit struct {
	ChunkID         int64
	KnowledgeBaseID int64
	ChunkIndex      int
	Content         string
	// Distance 为 pgvector 余弦距离（<=>）；越小越相似，取值约 [0,2]。
	Distance float64
}

// KnowledgeVectorSearcher 对 knowledge_base_chunks 做相似向量检索（POST /api/knowledgebase/query* 共用）。
type KnowledgeVectorSearcher interface {
	// SearchSimilarChunks 在指定知识库 id 集合内，按 queryEmbedding 做余弦距离升序取至多 limit 条。
	// kbIDs 须已由上层校验为非空且为正整数。
	SearchSimilarChunks(ctx context.Context, kbIDs []int64, queryEmbedding []float32, limit int) ([]KnowledgeChunkHit, error)
}

// KnowledgeBaseQueryChat OpenAI 兼容 Chat Completions：非流式整答与流式增量（由 infrastructure/ai 注入）。
type KnowledgeBaseQueryChat interface {
	// Complete systemPrompt + userPrompt 一次返回全文。
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	// Stream systemPrompt + userPrompt；对每个增量正文片段调用 onDelta（可为空串，调用方忽略）。
	Stream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(fragment string) error) error
}
