package repository

import (
	"context"
	"io"
)

type ObjectStoragePort interface {
	// 上传文件
	Upload(ctx context.Context, key string, body io.Reader, contentType string) error
	// 拉取整对象正文（S3/MinIO GetObject）；重分析时当 resume_text 为空用于重新解析
	GetObject(ctx context.Context, key string) (content []byte, contentType string, err error)
	// 删除文件
	DeleteObject(ctx context.Context, key string) error
	// 获取文件URL
	GetObjectPresignedURL(ctx context.Context, key string) (string, error)
}
