package results

// SessionRecordForSubmit 含 interview_sessions 主键与出题 JSON 快照，供提交答案时写 interview_answers、更新游标与刷 Redis。
type SessionRecordForSubmit struct {
	InternalID int64
	SessionID  string
	ResumeID   int64
	ResumeText string

	QuestionsJSON        string
	CurrentQuestionIndex int
	Status               string
	TotalQuestions       *int
}
