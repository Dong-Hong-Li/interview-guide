package results

// CurrentQuestionResponse GET /api/interview/sessions/{sessionId}/question 的响应体：
// completed 表示是否已结束，question 为当前应答题，message 用于无题时的提示文案。
type CurrentQuestionResponse struct {
	Completed bool `json:"completed"`
	// Question 当前应答题；已完成或无数时为空。
	Question *InterviewQuestion `json:"question,omitempty"`
	Message  string             `json:"message,omitempty"`
}
