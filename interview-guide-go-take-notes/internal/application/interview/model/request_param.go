package model

// CreateInterviewSessionReq 创建模拟面试会话请求参数
// POST /api/interview/sessions
type CreateInterviewSessionReq struct {
	// 简历文本
	ResumeText string `json:"resumeText"`
	// 题目数量
	QuestionCount int `json:"questionCount"`
	// 简历ID
	ResumeID *int64 `json:"resumeId,omitempty"`
	// 是否强制创建
	ForceCreate bool `json:"forceCreate,omitempty"`
}

// FindUnfinishedReq 按简历 ID 查询未结束的面试会话
// GET /api/interview/sessions/unfinished/{resumeId}
type FindUnfinishedReq struct {
	ResumeID int64 `path:"resumeId"`
}

// GetCurrentQuestionReq 轮询或拉取当前会话应答题
// GET /api/interview/sessions/{sessionId}/question
type GetCurrentQuestionReq struct {
	SessionID string `path:"sessionId"`
}

// DeleteInterviewSessionReq 删除一场面试及侧存
// DELETE /api/interview/sessions/{sessionId}
type DeleteInterviewSessionReq struct {
	SessionID string `path:"sessionId"`
}

// SubmitAnswerReq POST /api/interview/sessions/{sessionId}/answers
// JSON 体与前端 SubmitAnswerRequest 一致：每请求提交一题，questionIndex 为题号（0 表示第 1 题，可为 0 勿用 validate:"required" 绑整型）。
type SubmitAnswerReq struct {
	SessionID string `path:"sessionId" validate:"required"`
	// 与 session.questions 中下标、InterviewQuestion.questionIndex 一致
	QuestionIndex int    `json:"questionIndex"`
	Answer        string `json:"answer" validate:"required"`
}

// GetReportReq GET /api/interview/sessions/{sessionId}/report
type GetReportReq struct {
	SessionID string `path:"sessionId"`
}

// GetInterviewDetailReq GET /api/interview/sessions/{sessionId}/details
type GetInterviewDetailReq struct {
	SessionID string `path:"sessionId"`
}

// GetSessionReq GET /api/interview/sessions/{sessionId} 拉取会话详情（题目、游标、状态）。
type GetSessionReq struct {
	SessionID string `path:"sessionId"`
}

// CompleteSessionReq POST /api/interview/sessions/{sessionId}/complete 提前交卷，无 body。
type CompleteSessionReq struct {
	SessionID string `path:"sessionId"`
}

// ListInterviewSessionsReq GET /api/interview/sessions?page=&size=
type ListInterviewSessionsReq struct {
	Page     int `query:"page"`
	Size     int `query:"size"`
	PageSize int `query:"pageSize"`
}

// SaveAnswerReq PUT /api/interview/sessions/{sessionId}/answers — 与 POST 同路径，仅保存草稿、不跳题、不交卷、不触评估入队。
type SaveAnswerReq struct {
	SessionID string `path:"sessionId" validate:"required"`
	// 与 questions 下标一致
	QuestionIndex int    `json:"questionIndex"`
	Answer        string `json:"answer"`
}
