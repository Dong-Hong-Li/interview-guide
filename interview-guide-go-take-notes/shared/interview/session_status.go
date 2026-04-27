// Package interview 存放面试域与 Java 服务对齐的跨层常量（与 internal/application/.../model 解耦）。
package interview

// SessionStatus 与 Java InterviewSessionDTO.SessionStatus 一致，供 API、Redis/DB 复用。
type SessionStatus string

const (
	// StatusCreated 会话已创建
	StatusCreated SessionStatus = "CREATED"
	// StatusInProgress 面试进行中
	StatusInProgress SessionStatus = "IN_PROGRESS"
	// StatusCompleted 面试已完成
	StatusCompleted SessionStatus = "COMPLETED"
	// StatusEvaluated 已生成评估报告
	StatusEvaluated SessionStatus = "EVALUATED"
)
