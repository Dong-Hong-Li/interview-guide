package adapter

import (
	"context"
	"fmt"
	"interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/logmsg"
	"interview-guide-go/shared/streamkey"
	"strconv"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// KnowledgeVectorizePublisher 投递知识库向量化任务到 Redis Stream。
type KnowledgeVectorizePublisher struct {
	rdb *redis.Client
	lg  *zap.Logger
}

// NewKnowledgeVectorizePublisher Wire 注入。
func NewKnowledgeVectorizePublisher(rdb *redis.Client, lg *zap.Logger) repository.VectorizeTaskPublisher {
	return &KnowledgeVectorizePublisher{rdb: rdb, lg: lg}
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
	id, err := p.rdb.XAdd(ctx, args).Result()
	if err != nil {
		return err
	}
	if p.lg != nil {
		p.lg.Info(logmsg.MsgKnowledgeVectorizeEnqueued,
			zap.Int64("kbId", kbID),
			zap.String("stream", streamkey.StreamKnowledgeVectorize),
			zap.String(logmsg.FieldID, id),
			zap.Int("contentRunes", utf8.RuneCountInString(content)),
		)
	}
	return nil
}
