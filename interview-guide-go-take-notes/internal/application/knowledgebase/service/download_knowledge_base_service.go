package service

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// DownloadKnowledgeBaseService GET /api/knowledgebase/{id}/download 从对象存储拉取原始文件。
type DownloadKnowledgeBaseService struct {
	Reader  kbrepo.KnowledgeBaseReader
	Storage kbrepo.ObjectStoragePort
}

// NewDownloadKnowledgeBaseService Wire 注入。
func NewDownloadKnowledgeBaseService(
	r kbrepo.KnowledgeBaseReader,
	s kbrepo.ObjectStoragePort,
) *DownloadKnowledgeBaseService {
	return &DownloadKnowledgeBaseService{Reader: r, Storage: s}
}

// DownloadKnowledgeBaseResult 写 HTTP 响应用。
type DownloadKnowledgeBaseResult struct {
	Data        []byte
	ContentType string
	Filename    string
}

// DownloadFile 按 id 取对象；不存在 404，无 key 或存储未配置 4xx/5xx。
func (s *DownloadKnowledgeBaseService) DownloadFile(ctx context.Context, id int64) (*DownloadKnowledgeBaseResult, error) {
	if s == nil || s.Reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseDownloadServiceNil)
	}
	if s.Storage == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseObjectStorageNotConfigured)
	}
	if id < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	ref, err := s.Reader.GetKnowledgeBaseFileRef(ctx, id)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
	}
	key := strings.TrimSpace(ref.StorageKey)
	if key == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseNoStorageKey)
	}
	data, objectCT, err := s.Storage.GetObject(ctx, key)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetKnowledgeBaseObjectFailed+err.Error())
	}
	ct := strings.TrimSpace(objectCT)
	if ct == "" {
		ct = strings.TrimSpace(ref.ContentType)
	}
	if ct == "" {
		ct = errmsg.ApplicationOctetStream
	}
	return &DownloadKnowledgeBaseResult{
		Data:        data,
		ContentType: ct,
		Filename:    downloadAttachmentFilename(ref.OriginalFilename),
	}, nil
}

func downloadAttachmentFilename(original string) string {
	base := filepath.Base(strings.TrimSpace(original))
	if base == "" || base == "." || base == "/" {
		return "knowledge-base"
	}
	return base
}
