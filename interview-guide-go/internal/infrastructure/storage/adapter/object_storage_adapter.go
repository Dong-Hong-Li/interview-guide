// Package adapter 提供简历模块的存储适配器：把 S3/MinIO 的 *StorageService
// 接到 application/repository 的 ObjectStoragePort，使上传简历原件、
// 重分析时按 key 取回正文、删简历时删对象、导出/下载时出预签名链路由同一套 key 约定贯穿。
package adapter

import (
	"context"
	"errors"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/infrastructure/storage"
	"io"
)

// ObjectStorageAdapter 内聚一条 *StorageService，对简历用例做 Upload/Get/Delete/Presign 转发。
type ObjectStorageAdapter struct {
	SVC *storage.StorageService
}

// NewObjectStorageAdapter 在 Wire/deps 里用已连桶的 *StorageService 实现 ObjectStoragePort，
// 供简历上传、重分析、删除、导出等 service 注入。
func NewObjectStorageAdapter(svc *storage.StorageService) repository.ObjectStoragePort {
	return &ObjectStorageAdapter{SVC: svc}
}

// Upload 将用户上传的简历/附件以业务约定的 object key 写入桶（如 PDF 原文件上链）。
func (s *ObjectStorageAdapter) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if s.SVC == nil || key == "" || body == nil || contentType == "" {
		return errors.New("invalid parameters")
	}
	return s.SVC.Upload(ctx, key, body, contentType)
}

// GetObject 按 key 从桶里拉整文件（重分析时 resume 文本为空，用同一 key 再取 PDF 等原件解析正文）。
// 返回的 contentType 可能为空，上层可回退为库里存的简历 content_type。
func (s *ObjectStorageAdapter) GetObject(ctx context.Context, key string) ([]byte, string, error) {
	if s.SVC == nil {
		return nil, "", errors.New("storage service is not initialized")
	}
	if key == "" {
		return nil, "", errors.New("key is empty")
	}
	return s.SVC.GetObject(ctx, key)
}

// DeleteObject 按 key 从桶中删除与简历/附件对应的对象，与「删简历」等业务动作对齐。
func (s *ObjectStorageAdapter) DeleteObject(ctx context.Context, key string) error {
	if s.SVC == nil {
		return errors.New("storage service is not initialized")
	}
	if key == "" {
		return errors.New("invalid parameters")
	}
	return s.SVC.DeleteObject(ctx, key)
}

// GetObjectPresignedURL 生成带过期时间的直链，供下载简历原件、导出分析 PDF 等只读场景。
func (s *ObjectStorageAdapter) GetObjectPresignedURL(ctx context.Context, key string) (string, error) {
	if s.SVC == nil {
		return "", errors.New("storage service is not initialized")
	}
	if key == "" {
		return "", errors.New("key is empty")
	}
	return s.SVC.PresignGetObjectURL(ctx, key)
}
