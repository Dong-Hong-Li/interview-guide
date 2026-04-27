package service

import (
	"context"
	result "interview-guide-go/internal/application/resume/model/results"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/domain"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ResumeListService 简历列表服务（只返回 application/model/results，不依赖 infrastructure ORM 类型）。
type ResumeListService struct {
	ResumeWriter repository.ResumeWriter
	Logger       *zap.Logger
}

func NewResumeListService(logger *zap.Logger, resumeWriter repository.ResumeWriter) *ResumeListService {
	return &ResumeListService{ResumeWriter: resumeWriter, Logger: logger}
}

// ListResumes 分页获取简历列表；与前端 `history.ResumeListPage` 形状一致（content + 分页元信息）。
func (s *ResumeListService) ListResumes(ctx context.Context, page, size int) (*result.ResumeListResult, error) {
	if s.ResumeWriter == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.ResumePersistenceNotConfigured)
	}
	page, size = domain.NormalizeListPaging(page, size)
	rows, total, err := s.ResumeWriter.ListResumes(ctx, page, size)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.FailedListResumes+err.Error())
	}

	items := make([]result.ResumeListItem, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		item := result.ResumeListItem{
			ID:             r.ID,
			Filename:       r.OriginalFilename,
			FileSize:       r.FileSize,
			UploadedAt:     r.UploadedAt.UTC().Format(time.RFC3339Nano),
			AccessCount:    r.AccessCount,
			InterviewCount: 0, // 未接 interview 聚合时置 0，与主站有统计时的区别见后续查询
			AnalyzeStatus:  r.AnalyzeStatus,
		}
		// 简历分析失败原因
		if r.AnalyzeError != "" {
			item.AnalyzeError = r.AnalyzeError
		}
		if r.StorageURL != "" {
			item.StorageURL = strings.TrimSpace(r.StorageURL)
		}
		items = append(items, item)
	}
	return buildResumeListPage(items, total, page, size), nil
}

// GetStatistics 简历总数、面试会话总数、全库访问次数之和。
func (s *ResumeListService) GetStatistics(ctx context.Context) (*result.ResumeStatsResult, error) {
	if s.ResumeWriter == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.ResumePersistenceNotConfigured)
	}
	tc, tic, tac, err := s.ResumeWriter.AggregateResumeGlobalStats(ctx)
	if err != nil {
		return nil, response.Err(http.StatusInternalServerError, errmsg.FailedGetResumeStatistics+err.Error())
	}
	return &result.ResumeStatsResult{
		TotalCount:          tc,
		TotalInterviewCount: tic,
		TotalAccessCount:    tac,
	}, nil
}

// buildResumeListPage 构建简历列表分页结果。
func buildResumeListPage(content []result.ResumeListItem, total int64, page, size int) *result.ResumeListResult {
	totalPages := 0
	if total > 0 && size > 0 {
		totalPages = int((total + int64(size) - 1) / int64(size))
	}
	first := page <= 1 || total == 0
	last := totalPages == 0 || page >= totalPages
	return &result.ResumeListResult{
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
