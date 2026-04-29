package results

// SubmitAnswerResponse 与前端 `SubmitAnswerResponse` 一致。
type SubmitAnswerResponse struct {
	// 是否有下一题
	HasNextQuestion bool `json:"hasNextQuestion"`
	// 下一题
	NextQuestion *InterviewQuestion `json:"nextQuestion"`
	// 当前题索引
	CurrentIndex int `json:"currentIndex"`
	// 总题数
	TotalQuestions int `json:"totalQuestions"`
}
