// Package resume 存放简历域与存储/队列对齐的跨层常量（与 internal/application/.../model 解耦）。
package resume

// AnalyzeStatus 与 resumes.analyze_status、Java 侧及前端展示一致（队列消费者与落库复用）。
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
