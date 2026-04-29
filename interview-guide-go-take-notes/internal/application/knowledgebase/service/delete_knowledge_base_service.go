package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// DeleteKnowledgeBaseService DELETE /api/knowledgebase/{id}：先删对象存储再删库行；任一步失败即返回错误。
type DeleteKnowledgeBaseService struct {
	Reader  kbrepo.KnowledgeBaseReader
	Writer  kbrepo.KnowledgeBaseWriter
	Storage kbrepo.ObjectStoragePort
}

// NewDeleteKnowledgeBaseService Wire 注入。
func NewDeleteKnowledgeBaseService(
	r kbrepo.KnowledgeBaseReader,
	w kbrepo.KnowledgeBaseWriter,
	s kbrepo.ObjectStoragePort,
) *DeleteKnowledgeBaseService {
	return &DeleteKnowledgeBaseService{Reader: r, Writer: w, Storage: s}
}

// Delete 删除知识库；不存在 404，并发下已被删 404。
func (s *DeleteKnowledgeBaseService) Delete(ctx context.Context, id int64) error {
	if s == nil || s.Reader == nil || s.Writer == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseDeleteServiceNil)
	}
	if id < 1 {
		return response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	ref, err := s.Reader.GetKnowledgeBaseFileRef(ctx, id)
	if err != nil {
		return err
	}
	if ref == nil {
		return response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
	}
	if s.Storage == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseObjectStorageNotConfigured)
	}
	if key := strings.TrimSpace(ref.StorageKey); key != "" {
		if err := s.Storage.DeleteObject(ctx, key); err != nil {
			return response.Err(http.StatusBadGateway, errmsg.DeleteKnowledgeBaseObjectFailed+err.Error())
		}
	}
	if err := s.Writer.DeleteKnowledgeBaseByID(ctx, id); err != nil {
		if errors.Is(err, kbrepo.ErrKnowledgeBaseDeleteNoRow) {
			return response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
		}
		return err
	}
	return nil
}
