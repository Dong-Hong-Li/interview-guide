package model

// WorkerSession 评估消费者按对外 sessionId 加载会话时所需的字段子集（含内部主键，方便后续按 PK 写入）。
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
