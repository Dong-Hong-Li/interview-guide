package results

import "time"

// InterviewDetail GET /api/interview/sessions/{sessionId}/details 的响应数据体，供前端历史详情页直接消费。
type InterviewDetail struct {
	ID              int64      `json:"id"`
	SessionID       string     `json:"sessionId"`
	TotalQuestions  *int       `json:"totalQuestions"`
	Status          string     `json:"status"`
	EvaluateStatus  string     `json:"evaluateStatus,omitempty"`
	EvaluateError   string     `json:"evaluateError,omitempty"`
	OverallScore    *int       `json:"overallScore"`
	OverallFeedback string     `json:"overallFeedback"`
	CreatedAt       time.Time  `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt"`
	// Questions 为题面 JSON 数组原样；单题与答案合并见 Answers。
	Questions        []any                 `json:"questions"`
	Strengths        []string              `json:"strengths"`
	Improvements     []string              `json:"improvements"`
	ReferenceAnswers []any                 `json:"referenceAnswers"`
	Answers          []InterviewAnswerItem `json:"answers"`
}

// InterviewAnswerItem 与 dto.InterviewAnswerItem 字段一致。
type InterviewAnswerItem struct {
	QuestionIndex   int        `json:"questionIndex"`
	Question        string     `json:"question"`
	Category        string     `json:"category"`
	UserAnswer      string     `json:"userAnswer"`
	Score           int        `json:"score"`
	Feedback        string     `json:"feedback"`
	ReferenceAnswer string     `json:"referenceAnswer,omitempty"`
	KeyPoints       []string   `json:"keyPoints,omitempty"`
	AnsweredAt      *time.Time `json:"answeredAt,omitempty"`
}
