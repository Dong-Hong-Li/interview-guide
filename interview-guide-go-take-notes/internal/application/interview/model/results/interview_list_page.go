package results

import "time"

// InterviewListItem 对应前端 InterviewListRow 单行（与 history.ts 一致）。
type InterviewListItem struct {
	ID              int64      `json:"id"`
	SessionID       string     `json:"sessionId"`
	ResumeID        int64      `json:"resumeId"`
	ResumeFilename  string     `json:"resumeFilename"`
	TotalQuestions  int        `json:"totalQuestions"`
	Status          string     `json:"status"`
	EvaluateStatus  string     `json:"evaluateStatus,omitempty"`
	EvaluateError   string     `json:"evaluateError,omitempty"`
	OverallScore    *int       `json:"overallScore"`
	OverallFeedback *string    `json:"overallFeedback"`
	CreatedAt       time.Time  `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt"`
}

// InterviewListPage 对应前端 InterviewListPage（分页体）。
type InterviewListPage struct {
	Content       []InterviewListItem `json:"content"`
	TotalElements int64               `json:"totalElements"`
	TotalPages    int                 `json:"totalPages"`
	Page          int                 `json:"page"`
	Size          int                 `json:"size"`
	First         bool                `json:"first"`
	Last          bool                `json:"last"`
	HasNext       bool                `json:"hasNext"`
	HasPrevious   bool                `json:"hasPrevious"`
}
