package postgres

import (
	"context"
	"errors"
	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresService struct {
	DB *gorm.DB
}

func StartPostgresService(ctx context.Context, c *config.Config) (*PostgresService, error) {
	if c == nil || !c.Database.DatabaseConfigured() {
		return nil, errors.New(errmsg.ConfigPostgresHostRequired)
	}

	// 打开 Postgres
	db, err := gorm.Open(postgres.Open(c.Database.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 返回 Postgres 服务
	return &PostgresService{
		DB: db,
	}, nil
}

// Close 关闭 Postgres 服务
func (s *PostgresService) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
