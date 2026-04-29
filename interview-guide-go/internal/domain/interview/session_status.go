// Package interview 会话/评估状态常量与门禁，与主项目 internal/domain/interview 及 DB 列一致。
package interview

// 会话状态（与 interview_sessions.status、Redis、Java SessionStatus 字面量一致）。
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
