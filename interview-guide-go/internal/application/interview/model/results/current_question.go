package results

// CurrentQuestionVO 对应 GET /api/interview/sessions/{sessionId}/question 响应体，
// 与主项目 dto.CurrentQuestionResponse、前端 CurrentQuestionResponse 字段一致（completed / question / message）。
type CurrentQuestionResponse struct {
	Completed bool `json:"completed"`
	// Question 当前应答题；已完成或无数时为空。
	Question *InterviewQuestion `json:"question,omitempty"`
	Message  string             `json:"message,omitempty"`
}
