// Package mapper 直接实现 application/repository.ResumeWriter 端口（Mapper 即 Adapter），
// 不再引入单独的 ResumeWriterAdapter 样板层；GORM 模型只在本包内出现，对外暴露应用层 DTO。
package mapper

import (
	"context"
	"errors"
	"strings"

	ivrepo "interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/internal/application/resume/repository"
	model "interview-guide-go/internal/infrastructure/postgres/grom"

	"gorm.io/gorm"
)

// ResumeMapper 简历持久化适配器（GORM 实现）。
type ResumeMapper struct {
	gdb *gorm.DB
}

// NewResumeMapper 供 deps 组合根直接注入为 repository.ResumeWriter（*ResumeMapper 实现了该接口）。
func NewResumeMapper(gdb *gorm.DB) *ResumeMapper {
	return &ResumeMapper{gdb: gdb}
}

// 编译期断言：ResumeMapper 必须满足 repository.ResumeWriter。
var _ repository.ResumeWriter = (*ResumeMapper)(nil)

// 面试模块按 resume_id 读 interviewer_role 用于出题模板（backend/frontend）。
var _ ivrepo.InterviewerRoleReader = (*ResumeMapper)(nil)

var _ ivrepo.ResumeTextSource = (*ResumeMapper)(nil)

// ResumeTextAndInterviewerRole 供面试评估消费者拉取简历正文与人设（须含 resume_text）。
func (m *ResumeMapper) ResumeTextAndInterviewerRole(ctx context.Context, resumeID int64) (resumeText, interviewerRole string, err error) {
	row, err := m.getResumeRow(ctx, resumeID)
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(row.ResumeText), strings.TrimSpace(row.InterviewerRole), nil
}

// ── 写入 ────────────────────────────────────────────────────────────

// InsertResume 将应用层 ResumeInsert 映射为表行并插入，返回自增 id。
func (m *ResumeMapper) InsertResume(ctx context.Context, in *repository.ResumeInsert) (int64, error) {
	if in == nil {
		return 0, errors.New("resume insert: nil input")
	}
	sz := in.FileSize
	row := &model.Resume{
		FileHash:         in.FileHash,
		OriginalFilename: in.OriginalFilename,
		FileSize:         &sz,
		ContentType:      in.ContentType,
		StorageKey:       in.StorageKey,
		StorageURL:       in.StorageURL,
		ResumeText:       in.ResumeText,
		InterviewerRole:  in.InterviewerRole,
	}
	if err := m.gdb.WithContext(ctx).Create(row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// InsertResumeAnalysis 插入一条 resume_analyses 行。
func (m *ResumeMapper) InsertResumeAnalysis(ctx context.Context, in *repository.ResumeAnalysisInput) error {
	if in == nil {
		return errors.New("resume analysis insert: nil input")
	}
	row := &model.ResumeAnalysis{
		ResumeID:        in.ResumeID,
		OverallScore:    in.OverallScore,
		ContentScore:    in.ContentScore,
		StructureScore:  in.StructureScore,
		SkillMatchScore: in.SkillMatchScore,
		ExpressionScore: in.ExpressionScore,
		ProjectScore:    in.ProjectScore,
		Summary:         in.Summary,
		StrengthsJSON:   in.StrengthsJSON,
		SuggestionsJSON: in.SuggestionsJSON,
	}
	return m.gdb.WithContext(ctx).Create(row).Error
}

// InsertAnalyzeJob 占位：分析任务目前走 Redis Stream（AnalyzePublisher），无需落库。
func (m *ResumeMapper) InsertAnalyzeJob(_ context.Context, _ *repository.AnalyzeJob) error {
	return nil
}

// UpdateAnalyzeStatus 仅更新 analyze_status、analyze_error。
func (m *ResumeMapper) UpdateAnalyzeStatus(ctx context.Context, resumeID int64, status, analyzeErr string) error {
	return m.gdb.WithContext(ctx).Model(&model.Resume{}).Where("id = ?", resumeID).Updates(map[string]any{
		"analyze_status": status,
		"analyze_error":  analyzeErr,
	}).Error
}

// UpdateResumeText 仅更新 resume_text（重分析前从对象存储回灌纯文本缓存）。
func (m *ResumeMapper) UpdateResumeText(ctx context.Context, resumeID int64, resumeText string) error {
	if resumeID < 1 {
		return gorm.ErrRecordNotFound
	}
	return m.gdb.WithContext(ctx).Model(&model.Resume{}).Where("id = ?", resumeID).Update("resume_text", resumeText).Error
}

// ── 删除 ────────────────────────────────────────────────────────────

// DeleteResumeByID 按主键删除 resumes 行；0 行时返回 gorm.ErrRecordNotFound。
func (m *ResumeMapper) DeleteResumeByID(ctx context.Context, id int64) error {
	if id < 1 {
		return gorm.ErrRecordNotFound
	}
	res := m.gdb.WithContext(ctx).Where("id = ?", id).Delete(&model.Resume{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteResumeAnalysisByResumeID 删除某简历下全部 resume_analyses。
func (m *ResumeMapper) DeleteResumeAnalysisByResumeID(ctx context.Context, resumeID int64) error {
	if resumeID < 1 {
		return nil
	}
	return m.gdb.WithContext(ctx).Where("resume_id = ?", resumeID).Delete(&model.ResumeAnalysis{}).Error
}

// ── 查询 ────────────────────────────────────────────────────────────

// FindByFileHash 按 file_hash 去重查询；未命中返回 (nil, nil)，不触发 GORM 的 record-not-found 日志。
func (m *ResumeMapper) FindByFileHash(ctx context.Context, fileHash string) (*repository.ExistingResume, error) {
	if fileHash == "" {
		return nil, nil
	}
	var row model.Resume
	tx := m.gdb.WithContext(ctx).Where("file_hash = ?", fileHash).Limit(1).Find(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return toExistingResume(&row), nil
}

// GetResumeForAnalyze 分析消费者按 ID 加载简历；未命中返回 repository.ErrResumeNotFound。
func (m *ResumeMapper) GetResumeForAnalyze(ctx context.Context, resumeID int64) (*repository.ExistingResume, error) {
	row, err := m.getResumeRow(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	return toExistingResume(row), nil
}

// InterviewerRoleByResumeID 实现 interview.InterviewerRoleReader，供按简历选 backend/frontend 出题（与主项目模板一致）。
func (m *ResumeMapper) InterviewerRoleByResumeID(ctx context.Context, resumeID int64) (string, error) {
	ex, err := m.GetResumeForAnalyze(ctx, resumeID)
	if err != nil {
		return "", err
	}
	if ex == nil {
		return "FRONTEND", nil
	}
	r := strings.TrimSpace(ex.InterviewerRole)
	if r == "" {
		return "FRONTEND", nil
	}
	return r, nil
}

// GetResumeForDetail 详情页需要的全列简历头信息。
func (m *ResumeMapper) GetResumeForDetail(ctx context.Context, resumeID int64) (*repository.ResumeForDetail, error) {
	row, err := m.getResumeRow(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	var fs int64
	if row.FileSize != nil {
		fs = *row.FileSize
	}
	return &repository.ResumeForDetail{
		ID:               row.ID,
		OriginalFilename: row.OriginalFilename,
		FileSize:         fs,
		ContentType:      row.ContentType,
		StorageKey:       row.StorageKey,
		StorageURL:       row.StorageURL,
		ResumeText:       row.ResumeText,
		InterviewerRole:  row.InterviewerRole,
		AnalyzeStatus:    row.AnalyzeStatus,
		AnalyzeError:     row.AnalyzeError,
		UploadedAt:       row.UploadedAt,
		AccessCount:      row.AccessCount,
	}, nil
}

// ListAnalysesByResumeID 该简历下全部分析记录，按 analyzed_at 降序。
func (m *ResumeMapper) ListAnalysesByResumeID(ctx context.Context, resumeID int64) ([]repository.ResumeAnalysisListRow, error) {
	if resumeID < 1 {
		return []repository.ResumeAnalysisListRow{}, nil
	}
	var rows []model.ResumeAnalysis
	if err := m.gdb.WithContext(ctx).Where("resume_id = ?", resumeID).Order("analyzed_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]repository.ResumeAnalysisListRow, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, repository.ResumeAnalysisListRow{
			ID:              r.ID,
			OverallScore:    r.OverallScore,
			ContentScore:    r.ContentScore,
			StructureScore:  r.StructureScore,
			SkillMatchScore: r.SkillMatchScore,
			ExpressionScore: r.ExpressionScore,
			ProjectScore:    r.ProjectScore,
			Summary:         r.Summary,
			StrengthsJSON:   r.StrengthsJSON,
			SuggestionsJSON: r.SuggestionsJSON,
			AnalyzedAt:      r.AnalyzedAt,
		})
	}
	return out, nil
}

// ListResumes 分页查询简历表，按 uploaded_at 降序。
func (m *ResumeMapper) ListResumes(ctx context.Context, page, size int) ([]repository.ResumeListRow, int64, error) {
	var total int64
	if err := m.gdb.WithContext(ctx).Model(&model.Resume{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []repository.ResumeListRow{}, 0, nil
	}
	var rows []model.Resume
	err := m.gdb.WithContext(ctx).Model(&model.Resume{}).
		Order("uploaded_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]repository.ResumeListRow, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		var fs int64
		if r.FileSize != nil {
			fs = *r.FileSize
		}
		out = append(out, repository.ResumeListRow{
			ID:               r.ID,
			OriginalFilename: r.OriginalFilename,
			UploadedAt:       r.UploadedAt,
			FileSize:         fs,
			StorageURL:       r.StorageURL,
			AnalyzeStatus:    r.AnalyzeStatus,
			AnalyzeError:     r.AnalyzeError,
			InterviewerRole:  r.InterviewerRole,
			LastAccessedAt:   r.LastAccessedAt,
			AccessCount:      r.AccessCount,
		})
	}
	return out, total, nil
}

// AggregateResumeGlobalStats 全库简历条数、面试会话总数、简历 access_count 之和（PostgreSQL 方言）。
func (m *ResumeMapper) AggregateResumeGlobalStats(ctx context.Context) (totalCount, totalInterviewCount, totalAccessCount int64, err error) {
	var row resumeGlobalStatsRow
	err = m.gdb.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*)::bigint FROM resumes) AS total_count,
			(SELECT COUNT(*)::bigint FROM interview_sessions) AS total_interview_count,
			(SELECT COALESCE(SUM(access_count), 0)::bigint FROM resumes) AS total_access_count
	`).Scan(&row).Error
	if err != nil {
		return 0, 0, 0, err
	}
	return row.TotalCount, row.TotalInterviewCount, row.TotalAccessCount, nil
}

// ── 内部 ────────────────────────────────────────────────────────────

type resumeGlobalStatsRow struct {
	TotalCount          int64 `gorm:"column:total_count"`
	TotalInterviewCount int64 `gorm:"column:total_interview_count"`
	TotalAccessCount    int64 `gorm:"column:total_access_count"`
}

// getResumeRow 统一的按主键加载，未命中翻译为 repository.ErrResumeNotFound。
func (m *ResumeMapper) getResumeRow(ctx context.Context, id int64) (*model.Resume, error) {
	if id < 1 {
		return nil, repository.ErrResumeNotFound
	}
	var row model.Resume
	if err := m.gdb.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrResumeNotFound
		}
		return nil, err
	}
	return &row, nil
}

func toExistingResume(row *model.Resume) *repository.ExistingResume {
	if row == nil {
		return nil
	}
	return &repository.ExistingResume{
		ID:               row.ID,
		OriginalFilename: row.OriginalFilename,
		StorageKey:       row.StorageKey,
		StorageURL:       row.StorageURL,
		InterviewerRole:  row.InterviewerRole,
	}
}
