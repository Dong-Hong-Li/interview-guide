package service

import (
	"context"
	"encoding/json"
	result "interview-guide-go/internal/application/resume/model/results"
	"interview-guide-go/internal/application/resume/repository"
	"time"

	"go.uber.org/zap"
)

// ResumeDetailService 简历详情（GET /api/resumes/{id}/detail），结果形状与主站、前端 `ResumeDetail` 一致。
type ResumeDetailService struct {
	ResumeWriter repository.ResumeWriter
	Logger       *zap.Logger
}

// NewResumeDetailService 注入简历持久化端口；构造简历详情查询服务。
func NewResumeDetailService(
	logger *zap.Logger,
	resumeWriter repository.ResumeWriter,
) *ResumeDetailService {
	return &ResumeDetailService{
		ResumeWriter: resumeWriter,
		Logger:       logger,
	}
}

// GetResumeDetail 聚合简历头 + 分析历史；面试会话在 interview 未接线时返回空 interviews。
func (s *ResumeDetailService) GetResumeDetail(ctx context.Context, id int64) (*result.ResumeDetailResult, error) {
	// 1. 获取简历信息
	rec, err := s.ResumeWriter.GetResumeForDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. 获取分析历史
	rows, err := s.ResumeWriter.ListAnalysesByResumeID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. 结合分析历史和简历信息，返回简历详情
	analyses := make([]result.ResumeDetailAnalysis, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		strengths := unmarshalStringSlice(r.StrengthsJSON)
		sugg := unmarshalAnySlice(r.SuggestionsJSON)
		analyses = append(analyses, result.ResumeDetailAnalysis{
			ID:              r.ID,
			OverallScore:    intFromPtr(r.OverallScore),
			ContentScore:    intFromPtr(r.ContentScore),
			StructureScore:  intFromPtr(r.StructureScore),
			SkillMatchScore: intFromPtr(r.SkillMatchScore),
			ExpressionScore: intFromPtr(r.ExpressionScore),
			ProjectScore:    intFromPtr(r.ProjectScore),
			Summary:         r.Summary,
			AnalyzedAt:      r.AnalyzedAt.UTC().Format(time.RFC3339Nano),
			Strengths:       strengths,
			Suggestions:     sugg,
		})
	}

	// 4. 返回简历详情
	out := &result.ResumeDetailResult{
		ID:              rec.ID,
		Filename:        rec.OriginalFilename,
		FileSize:        rec.FileSize,
		ContentType:     rec.ContentType,
		StorageURL:      rec.StorageURL,
		UploadedAt:      rec.UploadedAt.UTC().Format(time.RFC3339Nano),
		AccessCount:     rec.AccessCount,
		ResumeText:      rec.ResumeText,
		InterviewerRole: rec.InterviewerRole,
		AnalyzeStatus:   rec.AnalyzeStatus,
		AnalyzeError:    rec.AnalyzeError,
		Analyses:        analyses,
		Interviews:      []result.ResumeDetailInterview{},
	}
	return out, nil
}

// 将指针转换为 int
func intFromPtr(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// 将 JSON 字符串转换为字符串切片
func unmarshalStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var s []string
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return []string{}
	}
	if s == nil {
		return []string{}
	}
	return s
}

// 将 JSON 字符串转换为 any 切片
func unmarshalAnySlice(raw string) []any {
	if raw == "" {
		return []any{}
	}
	var v []any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return []any{}
	}
	if v == nil {
		return []any{}
	}
	return v
}
