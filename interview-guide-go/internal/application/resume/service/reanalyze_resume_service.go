package service

import (
	"context"
	"fmt"
	"interview-guide-go/internal/application/resume/repository"
	sharedresume "interview-guide-go/shared/resume"
	"strings"

	"go.uber.org/zap"
)

// ReanalyzeResumeService 重新分析简历服务
type ReanalyzeResumeService struct {
	ResumeWriter     repository.ResumeWriter
	AnalyzePublisher repository.AnalyzePublisher
	Storage          repository.ObjectStoragePort
	TextExtractor    repository.TextExtractor
	Logger           *zap.Logger
}

func NewReanalyzeResumeService(
	resumeWriter repository.ResumeWriter,
	analyzePublisher repository.AnalyzePublisher,
	storage repository.ObjectStoragePort,
	textExtractor repository.TextExtractor,
	logger *zap.Logger,
) *ReanalyzeResumeService {
	return &ReanalyzeResumeService{
		ResumeWriter:     resumeWriter,
		AnalyzePublisher: analyzePublisher,
		Storage:          storage,
		TextExtractor:    textExtractor,
		Logger:           logger,
	}
}

// ReanalyzeResume 重新分析简历（与 example ResumeUploadService.reanalyze 对齐：可空库内文本时从对象存储拉取解析并写回，再 PENDING 并入队）。
func (s *ReanalyzeResumeService) ReanalyzeResume(ctx context.Context, id int64) error {
	// 1. 获取简历信息
	rec, err := s.ResumeWriter.GetResumeForDetail(ctx, id)
	if err != nil {
		return err
	}

	// 2. 检查简历文本
	resumeText := strings.TrimSpace(rec.ResumeText)
	if resumeText == "" {
		resumeText, err = s.rehydrateResumeTextFromStorage(ctx, rec)
		if err != nil {
			return err
		}
	}

	// 3. 更新简历分析状态
	if err = s.ResumeWriter.UpdateAnalyzeStatus(ctx, id, string(sharedresume.AnalyzeStatusPending), ""); err != nil {
		return err
	}

	// 4. 发送分析任务
	if err = s.AnalyzePublisher.SendAnalyzeTask(ctx, id, resumeText); err != nil {
		return err
	}
	return nil
}

// rehydrateResumeTextFromStorage 在 resume_text 为空时，
// 按 storage_key 下载并解析，并写回 DB（同 Java setResumeText + save 前的缓存回灌）。
func (s *ReanalyzeResumeService) rehydrateResumeTextFromStorage(ctx context.Context, rec *repository.ResumeForDetail) (string, error) {
	key := strings.TrimSpace(rec.StorageKey)
	if key == "" || s.Storage == nil {
		return "", repository.ErrResumeTextUnavailable
	}
	b, objectCT, err := s.Storage.GetObject(ctx, key)
	if err != nil {
		return "", fmt.Errorf("download resume from storage: %w", err)
	}
	ct := strings.TrimSpace(objectCT)
	if ct == "" {
		ct = strings.TrimSpace(rec.ContentType)
	}
	t := strings.TrimSpace(s.TextExtractor.ExtractResumeText(b, rec.OriginalFilename, ct))
	if t == "" {
		return "", repository.ErrResumeTextUnavailable
	}
	if err := s.ResumeWriter.UpdateResumeText(ctx, rec.ID, t); err != nil {
		return "", err
	}
	return t, nil
}
