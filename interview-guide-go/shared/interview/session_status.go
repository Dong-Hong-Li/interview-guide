// Package interview 存放面试域跨层使用的常量（与 internal/application/.../model 解耦），供 API、Redis、DB 复用。
package interview

// SessionStatus 面试会话状态字面量，对外 API、Redis 缓存与 DB 列共用。
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
