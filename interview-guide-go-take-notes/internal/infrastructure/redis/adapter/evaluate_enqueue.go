package adapter

import (
	"context"
	"fmt"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/streamkey"

	"github.com/redis/go-redis/v9"
)

// compile-time check
var _ repository.InterviewEvaluateEnqueuer = (*EvaluateEnqueue)(nil)

// EvaluateEnqueue 与 AnalyzePublisher 一致：在 adapter 内完成 XADD，实现 application 端口「投递面试评估任务」。
type EvaluateEnqueue struct {
	rdb *redis.Client
}

// NewEvaluateEnqueue 无 Redis 时 rdb 为空，Enqueue 为 no-op
func NewEvaluateEnqueue(rdb *redis.Client) *EvaluateEnqueue {
	return &EvaluateEnqueue{rdb: rdb}
}

func (a *EvaluateEnqueue) EnqueueInterviewEvaluate(ctx context.Context, sessionPublicID string) error {
	if a == nil || a.rdb == nil {
		return nil
	}
	sid := sessionPublicID
	if sid == "" {
		return fmt.Errorf("empty session id")
	}
	args := &redis.XAddArgs{
		Stream: streamkey.StreamInterviewEvaluate,
		Values: map[string]interface{}{
			streamkey.StreamFieldEvalSessionID:  sid,
			streamkey.StreamFieldEvalRetryCount: "0",
		},
	}
	return a.rdb.XAdd(ctx, args).Err()
}
