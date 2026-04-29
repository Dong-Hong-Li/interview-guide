package repository

import "context"

// KnowledgeChunkException 模型认定的不可入库片段（如乱码），仅记录与打日志，不进入向量分块。
type KnowledgeChunkException struct {
	RawExcerpt string `json:"raw_excerpt"`
	Reason     string `json:"reason"`
}

// KnowledgeChunkSplitResult AI 分片输出：可向量化的正文块 + 异常摘录列表。
type KnowledgeChunkSplitResult struct {
	Chunks     []string
	Exceptions []KnowledgeChunkException
}

// KnowledgeTextChunker 知识库全文分片（由 LLM 语义切分；与 Embeddings 分离）。
type KnowledgeTextChunker interface {
	// SplitForVectorize 将全文切为若干块；乱码等应进入 Exceptions，不得混入 Chunks。
	SplitForVectorize(ctx context.Context, fullText string) (KnowledgeChunkSplitResult, error)
}
