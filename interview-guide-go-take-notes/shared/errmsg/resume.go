package errmsg

// 简历应用层与 HTTP 适配层复用的业务文案（与 response.Err 或 ErrJSON 搭配）。
const (
	ResumeUploadServiceNil           = "简历上传服务未配置"
	ResumePersistenceNotConfigured   = "简历持久化未配置"
	ResumeUnsupportedContentType     = "不支持的文件类型"
	ResumeFileExceedsSizeLimit       = "简历文件超过大小限制"
	ResumeExtractTextEmpty           = "简历正文抽取失败：正文为空"
	DuplicateResumeSameFileHash      = "重复简历（相同文件哈希）"
	SaveResumeInvalidID              = "保存简历返回了无效的 ID"
	ResumeDetailServiceNotConfigured = "简历详情服务未配置"
	ResumeNotFound                   = "简历不存在"
	ResumeDeleteSuccess              = "简历删除成功"
	ResumeReanalyzeSuccess           = "简历重新分析成功"
	ExportServiceNotConfigured       = "导出服务未配置"
	InvalidResumeID                  = "无效的简历 ID"

	// PDFExportFontHint 中文字体缺失时的用户提示
	PDFExportFontHint = "PDF 导出需要配置中文字体：设置环境变量 RESUME_PDF_FONT_TTF 或安装系统 Noto/Arial Unicode 等字体"
)

// 与 err.Error() 拼接的固定前缀（末尾含 ": "）。
const (
	ValidateUploadResumeRequestFailed = "校验上传简历请求失败："
	FindExistingResumeFailed          = "查询已有简历失败："
	UploadFileFailed                  = "上传文件失败："
	GetFileURLFailed                  = "获取文件地址失败："
	SaveResumeFailed                  = "保存简历失败："
	SendAnalyzeTaskFailed             = "发送分析任务失败："
	FailedListResumes                 = "获取简历列表失败："
	FailedGetResumeStatistics         = "获取简历统计失败："
	FailedDeleteResume                = "删除简历失败："
	FailedReanalyzeResume             = "重新分析简历失败："
	PDFGenerateFailedPrefix           = "无法生成 PDF："
)
