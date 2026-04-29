package repository

import "context"

type kbVectorizeIDCtxKey struct{}

// ContextWithKnowledgeBaseVectorizeID 向 Redis Stream 向量化链路中的 Embed 调用注入 kbId，便于 Embedding HTTP 日志与知识库对齐。
func ContextWithKnowledgeBaseVectorizeID(ctx context.Context, kbID int64) context.Context {
	return context.WithValue(ctx, kbVectorizeIDCtxKey{}, kbID)
}

// KnowledgeBaseVectorizeIDFromContext 读取 ContextWithKnowledgeBaseVectorizeID 注入的 kbId；未注入时 ok 为 false。
func KnowledgeBaseVectorizeIDFromContext(ctx context.Context) (kbID int64, ok bool) {
	v := ctx.Value(kbVectorizeIDCtxKey{})
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}
