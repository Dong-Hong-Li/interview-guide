// Package resume 存放简历域与存储/队列对齐的跨层常量（与 internal/application/.../model 解耦）。
package resume

// AnalyzeStatus 简历分析状态字面量，与 resumes.analyze_status 列、Redis 消费者与前端展示共用。
type AnalyzeStatus string

const (
	// AnalyzeStatusPending 待处理
	AnalyzeStatusPending AnalyzeStatus = "PENDING"
	// AnalyzeStatusProcessing 处理中
	AnalyzeStatusProcessing AnalyzeStatus = "PROCESSING"
	// AnalyzeStatusCompleted 已完成
	AnalyzeStatusCompleted AnalyzeStatus = "COMPLETED"
	// AnalyzeStatusFailed 失败
	AnalyzeStatusFailed AnalyzeStatus = "FAILED"
)
