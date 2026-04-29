package controller

import (
	"context"
	"errors"
	"interview-guide-go/internal/application/resume/model"
	result "interview-guide-go/internal/application/resume/model/results"
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/application/resume/service"
	pdfexport "interview-guide-go/internal/infrastructure/pdf"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ResumeController 简历模块 HTTP 适配层：只负责路由与入参/出参，具体业务由注入的应用服务完成。
type ResumeController struct {
	// 上传简历
	UploadService *service.ResumeUploadService
	// 获取面试官角色
	InterviewerRolesService *service.InterviewerRolesService
	// 获取简历列表
	ResumeListService *service.ResumeListService
	// 删除简历
	ResumeDeleteService *service.ResumeDeleteService

	// 获取简历详情
	ResumeDetailService *service.ResumeDetailService

	// 重新分析简历
	ReanalyzeResumeService *service.ReanalyzeResumeService
	// 导出分析 PDF
	ExportAnalysisPDFService *service.ExportAnalysisPDFService
	// 日志
	Logger *zap.Logger
}

func (c *ResumeController) Register(r chi.Router) {
	r.Route(APIMountPath, func(sr chi.Router) {
		sr.Post(PathUpload, binding.Handle(c.uploadAndAnalyze))
		sr.Get(PathInterviewerRoles, binding.Exec(c.interviewerRoles))
		sr.Get(PathList, binding.Handle(c.listResumes))
		sr.Get(PathStatistics, binding.Exec(c.getStatistics))
		sr.Post(PathReanalyze, binding.Handle(c.reanalyzeResume))
		sr.Get(PathDetail, binding.Handle(c.getResumeDetail))
		sr.Get(PathExport, c.handleExportAnalysisPDF)
		sr.Delete(PathDelete, binding.Handle(c.deleteResume))
	})
}

// uploadAndAnalyze POST /api/resumes/upload — multipart/form-data 由 binding 填充 model.UploadResumeRequest。
func (c *ResumeController) uploadAndAnalyze(ctx context.Context, req model.UploadResumeRequest) (*result.UploadResumeResult, error) {
	return c.UploadService.UploadAndAnalyze(ctx, req)
}

// interviewerRoles GET /api/resumes/interviewer-roles
func (c *ResumeController) interviewerRoles(ctx context.Context) ([]model.InterviewerRoleOption, error) {
	return c.InterviewerRolesService.InterviewerRoles(ctx)
}

// listResumes GET /api/resumes/?page=&size=&pageSize=
func (c *ResumeController) listResumes(ctx context.Context, in model.ListResumeRequest) (*result.ResumeListResult, error) {
	size := in.Size
	if size == 0 {
		size = in.PageSize
	}
	return c.ResumeListService.ListResumes(ctx, in.Page, size)
}

// getStatistics GET /api/resumes/statistics
func (c *ResumeController) getStatistics(ctx context.Context) (*result.ResumeStatsResult, error) {
	return c.ResumeListService.GetStatistics(ctx)
}

// deleteResume DELETE /api/resumes/{id}
func (c *ResumeController) deleteResume(ctx context.Context, in model.IDPathRequest) (string, error) {
	err := c.ResumeDeleteService.DeleteResume(ctx, in.ID)
	if err != nil {
		return "", response.Err(http.StatusInternalServerError, errmsg.FailedDeleteResume+err.Error())
	}
	return errmsg.ResumeDeleteSuccess, nil
}

// getResumeDetail GET /api/resumes/{id}/detail
func (c *ResumeController) getResumeDetail(ctx context.Context, in model.IDPathRequest) (*result.ResumeDetailResult, error) {
	out, err := c.ResumeDetailService.GetResumeDetail(ctx, in.ID)
	if err != nil {
		if errors.Is(err, repository.ErrResumeNotFound) {
			return nil, response.Err(http.StatusInternalServerError, errmsg.ResumeNotFound)
		}
		return nil, err
	}
	return out, nil
}

// reanalyzeResume POST /api/resumes/{id}/reanalyze
func (c *ResumeController) reanalyzeResume(ctx context.Context, in model.IDPathRequest) (string, error) {
	err := c.ReanalyzeResumeService.ReanalyzeResume(ctx, in.ID)
	if err != nil {
		if errors.Is(err, repository.ErrResumeNotFound) {
			return "", response.Err(http.StatusNotFound, errmsg.ResumeNotFound)
		}
		if errors.Is(err, repository.ErrResumeTextUnavailable) {
			return "", response.Err(http.StatusBadRequest, err.Error())
		}
		return "", response.Err(http.StatusInternalServerError, errmsg.FailedReanalyzeResume+err.Error())
	}
	return errmsg.ResumeReanalyzeSuccess, nil
}

// handleExportAnalysisPDF GET /api/resumes/{id}/export：直接响应 application/pdf 二进制，不走 JSON binding。
func (c *ResumeController) handleExportAnalysisPDF(w http.ResponseWriter, r *http.Request) {
	if c.ExportAnalysisPDFService == nil {
		response.ErrJSON(w, http.StatusServiceUnavailable, errmsg.ExportServiceNotConfigured)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		response.ErrJSON(w, http.StatusBadRequest, errmsg.InvalidResumeID)
		return
	}
	out, err := c.ExportAnalysisPDFService.ExportAnalysisPDF(r.Context(), id)
	if err != nil {
		var he *response.Error
		if errors.As(err, &he) {
			response.ErrJSON(w, he.Code, he.Message)
			return
		}
		response.WriteErr(w, err)
		return
	}
	w.Header().Set("Content-Type", errmsg.ApplicationPDF)
	w.Header().Set("Content-Disposition", pdfexport.ContentDispositionRFC5987(out.Filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out.Bytes)
}
