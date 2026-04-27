package model

// WorkerSession 供评估 Redis 消费者按对外 sessionId 加载后的会话行形态（与主项目 InterviewSession 中 Worker 所需字段子集一致）。
type WorkerSession struct {
	ID int64
	// SessionID 对外的 session_id（UUID）
	SessionID string
	ResumeID  int64
	// Status 会话态：如 COMPLETED
	Status string
	// QuestionsJSON 题目 JSON
	QuestionsJSON string
	// EvaluateStatus 评估流水线状态
	EvaluateStatus string
}

// WorkerAnswer 仅消费者合并题目答案所需字段（按 session 主键查 interview_answers）。
type WorkerAnswer struct {
	QuestionIndex int
	UserAnswer    string
}
