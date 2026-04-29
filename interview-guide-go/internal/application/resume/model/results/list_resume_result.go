package result

// ResumeListItem 与 interview-guide 前端 `history.ResumeListItem` / 主服务 dto 对齐（GET /api/resumes 单行）。
type ResumeListItem struct {
	// 简历主键
	ID int64 `json:"id"`
	// 用户上传时的原始文件名
	Filename string `json:"filename"`
	// 文件大小（字节）
	FileSize int64 `json:"fileSize"`
	// 上传时间
	UploadedAt string `json:"uploadedAt"` // RFC3339
	// 访问次数
	AccessCount int `json:"accessCount"`
	// 最新评分
	LatestScore *int `json:"latestScore,omitempty"`
	// 最近分析时间
	LastAnalyzedAt string `json:"lastAnalyzedAt,omitempty"`
	// 面试次数
	InterviewCount int `json:"interviewCount"`
	// 简历分析状态
	AnalyzeStatus string `json:"analyzeStatus,omitempty"`
	// 简历分析失败原因
	AnalyzeError string `json:"analyzeError,omitempty"`
	// 可访问 URL（若有）
	StorageURL string `json:"storageUrl,omitempty"`
}

// ResumeListResult GET /api/resumes 成功体，与 `history.ResumeListPage`（content +	Spring 式分页元信息）一致。
type ResumeListResult struct {
	// 列表数据
	Content []ResumeListItem `json:"content"`
	// 总条数
	TotalElements int64 `json:"totalElements"`
	// 总页数
	TotalPages int `json:"totalPages"`
	// 当前页
	Page int `json:"page"`
	// 每页条数
	Size int `json:"size"`
	// 是否第一页
	First bool `json:"first"`
	// 是否最后一页
	Last bool `json:"last"`
	// 是否有下一页
	HasNext bool `json:"hasNext"`
	// 是否有上一页
	HasPrevious bool `json:"hasPrevious"`
}
