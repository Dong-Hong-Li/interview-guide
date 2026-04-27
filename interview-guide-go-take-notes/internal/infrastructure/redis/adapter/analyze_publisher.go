package adapter

import (
	"context"
	"fmt"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/shared/streamkey"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type AnalyzePublisher struct {
	rdb *redis.Client
}

func NewAnalyzePublisher(rdb *redis.Client) repository.AnalyzePublisher {
	return &AnalyzePublisher{rdb: rdb}
}

// 发送简历分析任务到 Redis Stream
func (p *AnalyzePublisher) SendAnalyzeTask(ctx context.Context, resumeID int64, resumeText string) error {
	if p.rdb == nil {
		return fmt.Errorf("nil redis client")
	}
	if resumeID < 1 {
		return fmt.Errorf("invalid resume id")
	}
	if resumeText == "" {
		return fmt.Errorf("invalid resume text")
	}
	args := &redis.XAddArgs{
		Stream: streamkey.StreamResumeAnalyze,
		Values: map[string]interface{}{
			streamkey.StreamFieldResumeID:   strconv.FormatInt(resumeID, 10),
			streamkey.StreamFieldContent:    resumeText,
			streamkey.StreamFieldRetryCount: "0",
		},
	}
	return p.rdb.XAdd(ctx, args).Err()
}
