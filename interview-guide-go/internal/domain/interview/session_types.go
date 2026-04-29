package interview

import "strings"

// SessionStatus 是会话状态字符串的标准化视图（大写、trim），与 DB / Redis 字面量一致。
type SessionStatus string

// ParseSessionStatus 将持久化或外部输入规范为 SessionStatus。
func ParseSessionStatus(raw string) SessionStatus {
	return SessionStatus(strings.ToUpper(strings.TrimSpace(raw)))
}

// QuestionsNotReady 表示题目尚未就绪（生成中或失败）。
func (s SessionStatus) QuestionsNotReady() bool {
	return s == SessionStatus(InterviewStatusQuestionsPending) || s == SessionStatus(InterviewStatusQuestionsFailed)
}

// AllowsAnswering 表示允许提交答案（会话未结束且题目流已就绪）。
func (s SessionStatus) AllowsAnswering() bool {
	return s == SessionStatus(InterviewStatusCreated) || s == SessionStatus(InterviewStatusInProgress)
}

// IsCompletedOrEvaluated 表示面试答题阶段已结束。
func (s SessionStatus) IsCompletedOrEvaluated() bool {
	return s == SessionStatus(InterviewStatusCompleted) || s == SessionStatus(InterviewStatusEvaluated)
}

// IsCompleted 表示状态为 COMPLETED（可能仍在评估中）。
func (s SessionStatus) IsCompleted() bool {
	return s == SessionStatus(InterviewStatusCompleted)
}

// EvaluateStatus 是评估子状态的标准化视图。
type EvaluateStatus string

// ParseEvaluateStatus 将 evaluate_status 列规范为 EvaluateStatus。
func ParseEvaluateStatus(raw string) EvaluateStatus {
	return EvaluateStatus(strings.ToUpper(strings.TrimSpace(raw)))
}

// IsCompleted 表示评估已成功完成。
func (e EvaluateStatus) IsCompleted() bool {
	return e == EvaluateStatus(InterviewEvaluateStatusCompleted)
}

// IsFailed 表示评估失败。
func (e EvaluateStatus) IsFailed() bool {
	return e == EvaluateStatus(InterviewEvaluateStatusFailed)
}
