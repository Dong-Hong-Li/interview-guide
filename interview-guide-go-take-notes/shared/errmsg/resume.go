package errmsg

// 简历应用层与 HTTP 适配层复用的业务文案（与 response.Err 或 ErrJSON 搭配）。
const (
	ResumeUploadServiceNil           = "resume upload service is nil"
	ResumePersistenceNotConfigured   = "resume persistence not configured"
	ResumeUnsupportedContentType     = "unsupported content type"
	ResumeFileExceedsSizeLimit       = "resume file exceeds size limit"
	ResumeExtractTextEmpty           = "extract resume text failed: empty text"
	DuplicateResumeSameFileHash      = "duplicate resume (same file hash)"
	SaveResumeInvalidID              = "save resume returned invalid id"
	ResumeDetailServiceNotConfigured = "resume detail service not configured"
	ResumeNotFound                   = "简历不存在"
	ResumeDeleteSuccess              = "简历删除成功"
	ResumeReanalyzeSuccess           = "简历重新分析成功"
	ExportServiceNotConfigured       = "export service not configured"
	InvalidResumeID                  = "无效的简历 ID"

	// PDFExportFontHint 中文字体缺失时的用户提示
	PDFExportFontHint = "PDF 导出需要配置中文字体：设置环境变量 RESUME_PDF_FONT_TTF 或安装系统 Noto/Arial Unicode 等字体"
)

// 与 err.Error() 拼接的固定前缀（末尾含 ": "）。
const (
	ValidateUploadResumeRequestFailed = "validate upload resume request failed: "
	FindExistingResumeFailed          = "find existing resume failed: "
	UploadFileFailed                  = "upload file failed: "
	GetFileURLFailed                  = "get file url failed: "
	SaveResumeFailed                  = "save resume failed: "
	SendAnalyzeTaskFailed             = "send analyze task failed: "
	FailedListResumes                 = "failed to list resumes: "
	FailedGetResumeStatistics         = "failed to get resume statistics: "
	FailedDeleteResume                = "failed to delete resume: "
	FailedReanalyzeResume             = "failed to reanalyze resume: "
	PDFGenerateFailedPrefix           = "无法生成 PDF："
)
