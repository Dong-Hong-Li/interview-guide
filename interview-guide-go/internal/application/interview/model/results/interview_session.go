package results

import "interview-guide-go/shared/interview"

// InterviewSession 与 Java InterviewSessionDTO 对齐：创建会话、恢复未完成会话、getSession 等 API 的 data 体。
type InterviewSession struct {
	SessionID            string                  `json:"sessionId"`
	ResumeID             int64                   `json:"resumeId,omitempty"`
	ResumeText           string                  `json:"resumeText"`
	TotalQuestions       int                     `json:"totalQuestions"`
	CurrentQuestionIndex int                     `json:"currentQuestionIndex"`
	Questions            []InterviewQuestion     `json:"questions"`
	Status               interview.SessionStatus `json:"status"`
}

// InterviewQuestion 与 Java InterviewQuestionDTO 字段一一对应。
// 新建会话时仅 index、question、type、category、追问标记有值，答题与评估后带 answer/score/feedback。
type InterviewQuestion struct {
	QuestionIndex       int                    `json:"questionIndex"`
	Question            string                 `json:"question"`
	Type                interview.QuestionType `json:"type"`
	Category            string                 `json:"category"` // 如：项目经历、Java 基础、集合、并发、MySQL、Redis、Spring、SpringBoot
	UserAnswer          *string                `json:"userAnswer,omitempty"`
	Score               *int                   `json:"score,omitempty"` // 0–100
	Feedback            *string                `json:"feedback,omitempty"`
	IsFollowUp          bool                   `json:"isFollowUp"`
	ParentQuestionIndex *int                   `json:"parentQuestionIndex,omitempty"` // 追问时指向主题索引
}
