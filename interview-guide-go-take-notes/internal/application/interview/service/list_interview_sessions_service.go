package service

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	domain "interview-guide-go/internal/domain"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// ListInterviewSessionsService GET /api/interview/sessions 分页列表（与前端 history.listInterviewSessions 一致）。
type ListInterviewSessionsService struct {
	sessions repository.InterviewSessionWriter
}

func NewListInterviewSessionsService(sessions repository.InterviewSessionWriter) *ListInterviewSessionsService {
	return &ListInterviewSessionsService{sessions: sessions}
}

// List page/size 在 controller 已合并 PageSize；此处再次 Normalize 防漏。
func (s *ListInterviewSessionsService) List(ctx context.Context, page, size int) (*results.InterviewListPage, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "list interview sessions not configured")
	}
	page, size = domain.NormalizeListPaging(page, size)
	rows, total, err := s.sessions.ListInterviewSessionsPage(ctx, page, size)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.GetInterviewSessionFailed)
	}
	return buildInterviewListPage(rows, total, page, size), nil
}

func buildInterviewListPage(content []results.InterviewListItem, total int64, page, size int) *results.InterviewListPage {
	totalPages := 0
	if total > 0 && size > 0 {
		totalPages = int((total + int64(size) - 1) / int64(size))
	}
	first := page <= 1 || total == 0
	last := totalPages == 0 || page >= totalPages
	return &results.InterviewListPage{
		Content:       content,
		TotalElements: total,
		TotalPages:    totalPages,
		Page:          page,
		Size:          size,
		First:         first,
		Last:          last,
		HasNext:       totalPages > 0 && page < totalPages,
		HasPrevious:   total > 0 && page > 1,
	}
}
