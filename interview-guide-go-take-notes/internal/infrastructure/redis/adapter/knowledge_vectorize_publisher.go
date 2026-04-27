package adapter

import (
	"context"
	"fmt"
	"interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/streamkey"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// KnowledgeVectorizePublisher 投递知识库向量化任务到 Redis Stream。
type KnowledgeVectorizePublisher struct {
	rdb *redis.Client
}

// NewKnowledgeVectorizePublisher Wire 注入。
func NewKnowledgeVectorizePublisher(rdb *redis.Client) repository.VectorizeTaskPublisher {
	return &KnowledgeVectorizePublisher{rdb: rdb}
}

// SendVectorizeTask 投递知识库向量化任务到 Redis Stream。
func (p *KnowledgeVectorizePublisher) SendVectorizeTask(ctx context.Context, kbID int64, content string) error {
	if p == nil || p.rdb == nil {
		return fmt.Errorf("redis client is not configured")
	}
	if kbID < 1 {
		return fmt.Errorf("invalid knowledge base id")
	}
	if content == "" {
		return fmt.Errorf("vectorize content is empty")
	}
	args := &redis.XAddArgs{
		Stream: streamkey.StreamKnowledgeVectorize,
		Values: map[string]interface{}{
			streamkey.StreamFieldKbID:       strconv.FormatInt(kbID, 10),
			streamkey.StreamFieldKbContent:  content,
			streamkey.StreamFieldRetryCount: "0",
		},
	}
	return p.rdb.XAdd(ctx, args).Err()
}
