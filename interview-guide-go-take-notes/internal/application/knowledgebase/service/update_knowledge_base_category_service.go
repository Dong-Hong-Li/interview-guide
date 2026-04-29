package service

import (
	"context"
	"errors"
	"net/http"

	"interview-guide-go/internal/application/knowledgebase/model"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// UpdateKnowledgeBaseCategoryService PUT /api/knowledgebase/{id}/category
type UpdateKnowledgeBaseCategoryService struct {
	reader kbrepo.KnowledgeBaseReader
	writer kbrepo.KnowledgeBaseWriter
}

// NewUpdateKnowledgeBaseCategoryService Wire 注入
func NewUpdateKnowledgeBaseCategoryService(
	r kbrepo.KnowledgeBaseReader,
	w kbrepo.KnowledgeBaseWriter,
) *UpdateKnowledgeBaseCategoryService {
	return &UpdateKnowledgeBaseCategoryService{reader: r, writer: w}
}

// Update 使用 controller 传入的 ValidatedKBUpdateCategory（已 trim/长度检查）。
func (s *UpdateKnowledgeBaseCategoryService) Update(ctx context.Context, in *model.ValidatedKBUpdateCategory) error {
	if s == nil || s.reader == nil || s.writer == nil {
		return response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseUpdateCategoryServiceNil)
	}
	if in == nil || in.ID < 1 {
		return response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	row, err := s.reader.GetKnowledgeBaseByID(ctx, in.ID)
	if err != nil {
		return err
	}
	if row == nil {
		return response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
	}
	if err := s.writer.UpdateKnowledgeBaseCategory(ctx, in.ID, in.Category); err != nil {
		if errors.Is(err, kbrepo.ErrKnowledgeBaseUpdateNoRow) {
			return response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
		}
		return err
	}
	return nil
}
