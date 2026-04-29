// Package interview 面试会话/评估的状态常量与门禁函数，集中管理避免业务包写状态字面量。
package interview

// 会话状态（与 interview_sessions.status、Redis 缓存中的状态字面量一致）。
const (
	InterviewStatusCreated          = "CREATED"
	InterviewStatusQuestionsPending = "QUESTIONS_PENDING"
	InterviewStatusQuestionsFailed  = "QUESTIONS_FAILED"
	InterviewStatusInProgress       = "IN_PROGRESS"
	InterviewStatusCompleted        = "COMPLETED"
	InterviewStatusEvaluated        = "EVALUATED"
)

// 评估状态（与 interview_sessions.evaluate_status 一致）。
const (
	InterviewEvaluateStatusPending    = "PENDING"
	InterviewEvaluateStatusProcessing = "PROCESSING"
	InterviewEvaluateStatusCompleted  = "COMPLETED"
	InterviewEvaluateStatusFailed     = "FAILED"
)
