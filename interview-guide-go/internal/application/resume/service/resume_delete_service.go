package service

import (
	"context"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/shared/logmsg"
	"strings"

	"go.uber.org/zap"
)

// ResumeDeleteService 简历删除（与 example Java：先删子数据再删简历行；对象存储失败仅告警不阻断）。
type ResumeDeleteService struct {
	ResumeWriter  repository.ResumeWriter
	ObjectStorage repository.ObjectStoragePort
	Logger        *zap.Logger
}

// NewResumeDeleteService ObjectStorage 可为 nil（未接线对象存储时仍删库）。
func NewResumeDeleteService(logger *zap.Logger, resumeWriter repository.ResumeWriter, objectStorage repository.ObjectStoragePort) *ResumeDeleteService {
	return &ResumeDeleteService{
		ResumeWriter:  resumeWriter,
		ObjectStorage: objectStorage,
		Logger:        logger,
	}
}

// DeleteResume 步骤：1) 取简历 2) 尽量删对象存储 3) 删 resume_analyses 4) 删 resumes。
// 与 Java `ResumeDeleteService.deleteResume` + `deleteResume` 事务内「先分析记录再实体」一致；面试会话删除待 interview 域接线后补充。
func (s *ResumeDeleteService) DeleteResume(ctx context.Context, id int64) error {
	rec, err := s.ResumeWriter.GetResumeForAnalyze(ctx, id)
	if err != nil {
		return err
	}

	// 1. 删除对象存储
	if s.ObjectStorage != nil {
		if key := strings.TrimSpace(rec.StorageKey); key != "" {
			if err := s.ObjectStorage.DeleteObject(ctx, key); err != nil {
				s.Logger.Warn(logmsg.MsgResumeDeleteStorageContinue, zap.String("key", key), zap.Error(err))
			}
		}
	}
	/* TODO: 删除面试相关会话 */

	// 4. 删除简历分析结果
	if err := s.ResumeWriter.DeleteResumeAnalysisByResumeID(ctx, id); err != nil {
		return err
	}

	// 5. 删除简历
	if err := s.ResumeWriter.DeleteResumeByID(ctx, id); err != nil {
		return err
	}
	return nil
}
