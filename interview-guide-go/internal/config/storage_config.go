package config

import (
	"errors"
	"os"
	"strconv"

	"interview-guide-go/shared/errmsg"
)

type StorageConfig struct {
	// storage 端点
	StorageEndpoint string
	// storage 访问密钥
	StorageAccessKey string
	// storage 密钥
	StorageSecretKey string
	// storage 桶
	StorageBucket string
	// storage 区域 默认 us-east-1
	StorageRegion string
	// storage 预签名 URL 有效期（秒），仅对象存储实现支持时生效。
	StoragePresignGetExpiresSec int
}

// 验证 storage 配置
func validateStorageConfig() (*StorageConfig, error) {
	// storage 配置
	storageEndpoint := os.Getenv("APP_STORAGE_ENDPOINT")
	if storageEndpoint == "" {
		return nil, errors.New(errmsg.ConfigStorageEndpointRequired)
	}
	// storage 访问密钥
	storageAccessKey := os.Getenv("APP_STORAGE_ACCESS_KEY")
	if storageAccessKey == "" {
		return nil, errors.New(errmsg.ConfigStorageAccessKeyRequired)
	}
	// storage 密钥
	storageSecretKey := os.Getenv("APP_STORAGE_SECRET_KEY")
	if storageSecretKey == "" {
		return nil, errors.New(errmsg.ConfigStorageSecretKeyRequired)
	}
	// storage 桶
	storageBucket := os.Getenv("APP_STORAGE_BUCKET")
	if storageBucket == "" {
		return nil, errors.New(errmsg.ConfigStorageBucketRequired)
	}
	// storage 区域
	storageRegion := os.Getenv("APP_STORAGE_REGION")
	if storageRegion == "" {
		return nil, errors.New(errmsg.ConfigStorageRegionRequired)
	}
	// storage 预签名 URL 有效期（秒），仅对象存储实现支持时生效。
	storagePresignGetExpiresSec, err := strconv.Atoi(os.Getenv("APP_STORAGE_PRESIGN_GET_EXPIRES_SEC"))
	if err != nil || storagePresignGetExpiresSec <= 0 {
		return nil, errors.New(errmsg.ConfigStoragePresignExpiresInvalid)
	}
	return &StorageConfig{
		StorageEndpoint:             storageEndpoint,
		StorageAccessKey:            storageAccessKey,
		StorageSecretKey:            storageSecretKey,
		StorageBucket:               storageBucket,
		StorageRegion:               storageRegion,
		StoragePresignGetExpiresSec: storagePresignGetExpiresSec,
	}, nil
}

// StorageConfigured 是否具备对象存储必填项。
func (c *StorageConfig) StorageConfigured() bool {
	return c.StorageEndpoint != "" && c.StorageAccessKey != "" && c.StorageSecretKey != "" && c.StorageBucket != "" && c.StorageRegion != "" && c.StoragePresignGetExpiresSec > 0
}
