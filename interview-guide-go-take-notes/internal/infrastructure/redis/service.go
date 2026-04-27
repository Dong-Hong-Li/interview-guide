package redis

import (
	"context"
	"errors"
	"fmt"
	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	Client *redis.Client
}

// StartRedisService 启动 Redis 服务
func StartRedisService(ctx context.Context, c *config.Config) (*RedisService, error) {
	// NewClient 由应用配置创建 Redis 客户端；未配置 REDIS_HOST 时返回 nil。
	if c == nil || !c.Redis.RedisConfigured() {
		return nil, errors.New(errmsg.ConfigRedisHostRequired)
	}
	return &RedisService{
		Client: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", c.Redis.RedisHost, c.Redis.RedisPort),
			Password: c.Redis.RedisPassword,
			DB:       c.Redis.RedisDB,
		}),
	}, nil
}

func (s *RedisService) Close() error {
	return s.Client.Close()
}
