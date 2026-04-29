package service

import (
	"context"
	"interview-guide-go/internal/application/resume/model"
	sharedresume "interview-guide-go/shared/resume"

	"go.uber.org/zap"
)

// InterviewerRolesService 面试官角色服务
type InterviewerRolesService struct {
	Logger *zap.Logger
}

func NewInterviewerRolesService(logger *zap.Logger) *InterviewerRolesService {
	return &InterviewerRolesService{Logger: logger}
}

// InterviewerRoles 固定面试官角色枚举（与 promptprofile 及上传入参一致，纯静态）。
func (s *InterviewerRolesService) InterviewerRoles(_ context.Context) ([]model.InterviewerRoleOption, error) {
	return []model.InterviewerRoleOption{
		{Value: string(sharedresume.InterviewerRoleBackend), Label: sharedresume.InterviewerRoleBackendLabel},
		{Value: string(sharedresume.InterviewerRoleFrontend), Label: sharedresume.InterviewerRoleFrontendLabel},
	}, nil
}
