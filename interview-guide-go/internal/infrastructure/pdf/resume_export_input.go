package pdfexport

import "time"

// ResumeExport 渲染简历分析 PDF 所需的简历元数据（基础设施层自有类型，不依赖 application）。
type ResumeExport struct {
	OriginalFilename string
	UploadedAt       time.Time
}

// ResumeAnalysisExport 渲染 PDF 所需的单次分析结果（指针语义与持久化层一致，nil 视为 0）。
type ResumeAnalysisExport struct {
	OverallScore    *int
	ProjectScore    *int
	SkillMatchScore *int
	ContentScore    *int
	StructureScore  *int
	ExpressionScore *int
	Summary         string
	StrengthsJSON   string
	SuggestionsJSON string
}
