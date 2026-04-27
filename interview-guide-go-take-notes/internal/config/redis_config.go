package config

import (
	"errors"
	"os"
	"strconv"

	"interview-guide-go/shared/errmsg"
)

type RedisConfig struct {
	// redis 主机
	RedisHost string
	// redis 端口
	RedisPort string
	// redis 数据库
	RedisDB int
	// redis 密码
	RedisPassword string
}

// 验证 redis 配置
func validateRedisConfig() (*RedisConfig, error) {
	// redis 配置
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		return nil, errors.New(errmsg.ConfigRedisHostRequired)
	}
	// redis 端口
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		return nil, errors.New(errmsg.ConfigRedisPortRequired)
	}
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil || redisDB < 0 {
		return nil, errors.New(errmsg.ConfigRedisDBInvalid)
	}
	// redis 密码（本地 compose 常见无密码，空字符串表示不传 AUTH）
	redisPassword := os.Getenv("REDIS_PASSWORD")

	return &RedisConfig{
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisDB:       redisDB,
		RedisPassword: redisPassword,
	}, nil
}

// RedisConfigured 是否启用 Redis（REDIS_HOST 非空）。
func (c *RedisConfig) RedisConfigured() bool {
	return c.RedisHost != "" || c.RedisPort != ""
}
