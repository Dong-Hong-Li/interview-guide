package result

// ResumeDetailResult 简历详情
type ResumeDetailResult struct {
	// 简历主键
	ID int64 `json:"id"`
	// 用户上传时的原始文件名
	Filename string `json:"filename"`
	// 文件大小（字节）
	FileSize int64 `json:"fileSize"`
	// 文件类型
	ContentType string `json:"contentType"`
	// 可访问 URL（若有）
	StorageURL string `json:"storageUrl"`
	// 上传时间
	UploadedAt string `json:"uploadedAt"`
	// 访问次数
	AccessCount int `json:"accessCount"`
	// 解析后的纯文本简历；面试出题主要依据
	ResumeText string `json:"resumeText"`
	// 与前端下拉 value 一致（如 BACKEND / FRONTEND），决定面试 prompts 人设
	InterviewerRole string `json:"interviewerRole,omitempty"`
	// 简历分析状态
	AnalyzeStatus string `json:"analyzeStatus,omitempty"`
	// 简历分析失败原因
	AnalyzeError string `json:"analyzeError,omitempty"`
	// 分析历史
	Analyses []ResumeDetailAnalysis `json:"analyses"`
	// 面试历史
	Interviews []ResumeDetailInterview `json:"interviews"`
}

// ResumeDetailAnalysis 简历分析历史
type ResumeDetailAnalysis struct {
	// 分析主键
	ID int64 `json:"id"`
	// 整体评分
	OverallScore int `json:"overallScore"`
	// 内容评分
	ContentScore int `json:"contentScore"`
	// 结构评分
	StructureScore int `json:"structureScore"`
	// 技能匹配评分
	SkillMatchScore int `json:"skillMatchScore"`
	// 表达评分
	ExpressionScore int `json:"expressionScore"`
	// 项目评分
	ProjectScore int `json:"projectScore"`
	// 总结
	Summary string `json:"summary"`
	// 分析时间
	AnalyzedAt string `json:"analyzedAt"`
	// 优势
	Strengths []string `json:"strengths"`
	// 建议
	Suggestions []any `json:"suggestions"`
}

// ResumeDetailInterview 面试历史
type ResumeDetailInterview struct {
	// 面试主键
	ID int64 `json:"id"`
	// 面试会话 ID
	SessionID string `json:"sessionId"`
	// 总问题数
	TotalQuestions int `json:"totalQuestions"`
	// 面试状态
	Status string `json:"status"`
	// 评估状态
	EvaluateStatus string `json:"evaluateStatus,omitempty"`
	// 评估失败原因
	EvaluateError string `json:"evaluateError,omitempty"`
	// 整体评分
	OverallScore int `json:"overallScore"`
	// 整体反馈
	OverallFeedback string `json:"overallFeedback"`
}
