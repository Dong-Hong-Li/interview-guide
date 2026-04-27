package config

import (
	"errors"
	"os"

	"interview-guide-go/shared/errmsg"
)

type DatabaseConfig struct {
	// database 配置
	DatabaseURL string
	// postgres 主机
	PGHost string
	// postgres 端口
	PGPort string
	// postgres 用户
	PGUser string
	// postgres 密码
	PGPassword string
	// postgres 数据库名称
	PGDBName string
	// postgres ssl 模式
	PGSSLMode string
}

// 验证 database 配置
func validateDatabaseConfig() (*DatabaseConfig, error) {
	// database 配置
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New(errmsg.ConfigDatabaseURLRequired)
	}
	// postgres 主机
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		return nil, errors.New(errmsg.ConfigPostgresHostRequired)
	}
	// postgres 端口
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		return nil, errors.New(errmsg.ConfigPostgresPortRequired)
	}
	// postgres 用户
	postgresUser := os.Getenv("POSTGRES_USER")
	if postgresUser == "" {
		return nil, errors.New(errmsg.ConfigPostgresUserRequired)
	}
	// postgres 密码
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	if postgresPassword == "" {
		return nil, errors.New(errmsg.ConfigPostgresPasswordRequired)
	}
	// postgres 数据库名称
	postgresDBName := os.Getenv("POSTGRES_DB")
	if postgresDBName == "" {
		return nil, errors.New(errmsg.ConfigPostgresDBNameRequired)
	}
	// postgres ssl 模式
	postgresSSLMode := os.Getenv("POSTGRES_SSLMODE")
	if postgresSSLMode == "" {
		return nil, errors.New(errmsg.ConfigPostgresSSLModeRequired)
	}
	return &DatabaseConfig{
		DatabaseURL: databaseURL,
		PGHost:      postgresHost,
		PGPort:      postgresPort,
		PGUser:      postgresUser,
		PGPassword:  postgresPassword,
		PGDBName:    postgresDBName,
		PGSSLMode:   postgresSSLMode,
	}, nil
}

// DatabaseConfigured 是否启用 Postgres（POSTGRES_HOST 非空）。
func (c *DatabaseConfig) DatabaseConfigured() bool {
	return c.DatabaseURL != "" || c.PGHost != "" || c.PGPort != ""
}
