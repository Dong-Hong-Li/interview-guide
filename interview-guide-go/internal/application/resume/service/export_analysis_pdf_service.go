package service

import (
	"context"
	"encoding/json"
	"errors"
	result "interview-guide-go/internal/application/resume/model/results"
	"interview-guide-go/internal/application/resume/repository"
	pdfexport "interview-guide-go/internal/infrastructure/pdf"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	sharedresume "interview-guide-go/shared/resume"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ExportAnalysisPDFService 导出简历分析 PDF。
// 只保留一个实现且仅被本服务使用，因此不再额外引入 PDFExportPort 端口与 pdfadapter 包；
// DTO 转换与字节流组装集中在本服务中。
type ExportAnalysisPDFService struct {
	ResumeDetailService *ResumeDetailService
	Logger              *zap.Logger
}

// NewExportAnalysisPDFService 注入详情服务；PDF 渲染由 infrastructure/pdf 直接完成。
func NewExportAnalysisPDFService(detail *ResumeDetailService, logger *zap.Logger) *ExportAnalysisPDFService {
	return &ExportAnalysisPDFService{ResumeDetailService: detail, Logger: logger}
}

// ExportAnalysisPDFResult GET /api/resumes/{id}/export 的二进制与下载文件名。
type ExportAnalysisPDFResult struct {
	Bytes    []byte
	Filename string
}

// ExportAnalysisPDF 加载详情并渲染 PDF；无分析记录或字体缺失时返回业务错误。
func (s *ExportAnalysisPDFService) ExportAnalysisPDF(ctx context.Context, id int64) (*ExportAnalysisPDFResult, error) {
	if s.ResumeDetailService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.ResumeDetailServiceNotConfigured)
	}
	detail, err := s.ResumeDetailService.GetResumeDetail(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrResumeNotFound) {
			return nil, response.Err(http.StatusNotFound, errmsg.ResumeNotFound)
		}
		return nil, err
	}
	pdfBytes, err := renderResumeAnalysisPDF(detail)
	if err != nil {
		if errors.Is(err, pdfexport.ErrNoFont) {
			return nil, response.Err(http.StatusServiceUnavailable, errmsg.PDFExportFontHint)
		}
		return nil, response.Err(http.StatusBadRequest, errmsg.PDFGenerateFailedPrefix+err.Error())
	}
	return &ExportAnalysisPDFResult{
		Bytes:    pdfBytes,
		Filename: pdfexport.ExportFilename(detail.Filename),
	}, nil
}

// renderResumeAnalysisPDF 把详情结果映射为 pdfexport DTO 再渲染。
// 仅使用详情里「最新一条分析记录」，与 Java ResumeHistoryService.exportAnalysisPdf 语义一致。
func renderResumeAnalysisPDF(d *result.ResumeDetailResult) ([]byte, error) {
	if d == nil || len(d.Analyses) == 0 {
		return nil, sharedresume.ErrNoResumeAnalysis
	}
	uploadedAt, err := time.Parse(time.RFC3339Nano, d.UploadedAt)
	if err != nil {
		if uploadedAt, err = time.Parse(time.RFC3339, d.UploadedAt); err != nil {
			uploadedAt = time.Time{}
		}
	}
	latest := d.Analyses[0]
	strengthsJSON, _ := json.Marshal(latest.Strengths)
	suggestionsJSON, _ := json.Marshal(latest.Suggestions)

	return pdfexport.RenderResumeAnalysisPDF(
		&pdfexport.ResumeExport{
			OriginalFilename: d.Filename,
			UploadedAt:       uploadedAt,
		},
		&pdfexport.ResumeAnalysisExport{
			OverallScore:    intPtrVal(latest.OverallScore),
			ProjectScore:    intPtrVal(latest.ProjectScore),
			SkillMatchScore: intPtrVal(latest.SkillMatchScore),
			ContentScore:    intPtrVal(latest.ContentScore),
			StructureScore:  intPtrVal(latest.StructureScore),
			ExpressionScore: intPtrVal(latest.ExpressionScore),
			Summary:         latest.Summary,
			StrengthsJSON:   string(strengthsJSON),
			SuggestionsJSON: string(suggestionsJSON),
		},
	)
}

// intPtrVal 返回指向 v 的副本指针（0 也保留，与评分语义一致）。
func intPtrVal(v int) *int {
	p := v
	return &p
}
