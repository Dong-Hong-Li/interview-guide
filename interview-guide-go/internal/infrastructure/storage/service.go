package storage

import (
	"context"
	"errors"
	"fmt"
	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService struct {
	// S3 客户端
	client *s3.Client
	// 存储桶名称
	bucket string
	// 预签名下载链接过期时间
	presignGetExpires time.Duration
	// 初始化存储桶
	initBucket sync.Once
	// 初始化错误
	initErr error
}

// StartStorageService 启动 S3 客户端（配置由 internal/config.LoadEnvironmentVariables 统一解析）。
func StartStorageService(ctx context.Context, c *config.Config) (*StorageService, error) {
	if c == nil || !c.Storage.StorageConfigured() {
		return nil, errors.New(errmsg.ConfigStorageEndpointRequired)
	}

	// 初始化S3 客户端配置
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(c.Storage.StorageRegion),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.Storage.StorageAccessKey, c.Storage.StorageSecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	//创建S3 客户端
	s3service := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(c.Storage.StorageEndpoint)
	})

	// 预签名下载链接过期时间
	sec := c.Storage.StoragePresignGetExpiresSec

	return &StorageService{
		client:            s3service,
		bucket:            c.Storage.StorageBucket,
		presignGetExpires: time.Duration(sec) * time.Second,
	}, nil
}

// Upload 上传文件到 S3
func (s *StorageService) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	s.ensureBucket(ctx) // 如果存储桶不存在，则创建存储桶
	if s.initErr != nil {
		return s.initErr
	}

	// 上传文件到 S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

// PresignGetObjectURL 生成短期下载链接（SigV4 预签名 GET）；与 MinIO/S3 兼容端点一致。
func (s *StorageService) PresignGetObjectURL(ctx context.Context, objectKey string) (string, error) {
	if objectKey == "" {
		return "", errors.New("empty object key")
	}
	pc := s3.NewPresignClient(s.client)
	out, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.presignGetExpires
	})
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}
	return out.URL, nil
}

// GetObject 从桶中拉取对象正文；用于重分析时按 storage_key 回灌 resume_text，避免再次解析 PDF。
func (s *StorageService) GetObject(ctx context.Context, key string) ([]byte, string, error) {
	if key == "" {
		return nil, "", errors.New("empty object key")
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("get object: %w", err)
	}
	defer out.Body.Close()
	data, rerr := io.ReadAll(out.Body)
	if rerr != nil {
		return nil, "", fmt.Errorf("read object body: %w", rerr)
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return data, ct, nil
}

// DeleteObject 从桶中删除对象；key 为空时为 no-op。
func (s *StorageService) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// ensureBucket 确保存储桶存在
func (s *StorageService) ensureBucket(ctx context.Context) {
	s.initBucket.Do(func() {
		// 检查存储桶是否存在
		_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(s.bucket),
		})
		if err == nil {
			return // 存储桶存在 直接返回
		}

		// 如果存储桶不存在，则创建存储桶
		_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.bucket),
		})

		// 创建存储桶失败 设置初始化错误
		if createErr != nil {
			s.initErr = fmt.Errorf("ensure bucket: %w", createErr)
		}

		// 创建存储桶成功 设置初始化错误为 nil
		s.initErr = nil
	})
}
