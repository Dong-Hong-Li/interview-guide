package results

import "time"

// InterviewReport GET /api/interview/sessions/{id}/report，与主项目、前端 `InterviewReport` 字段一致。

type InterviewReport struct {
	SessionID        string                   `json:"sessionId"`
	TotalQuestions   int                      `json:"totalQuestions"`
	OverallScore     int                      `json:"overallScore"`
	CategoryScores   []InterviewCategoryScore `json:"categoryScores"`
	QuestionDetails  []QuestionEvaluation     `json:"questionDetails"`
	OverallFeedback  string                   `json:"overallFeedback"`
	Strengths        []string                 `json:"strengths"`
	Improvements     []string                 `json:"improvements"`
	ReferenceAnswers []ReferenceAnswer        `json:"referenceAnswers"`
}

type InterviewCategoryScore struct {
	Category      string `json:"category"`
	Score         int    `json:"score"`
	QuestionCount int    `json:"questionCount"`
}

type QuestionEvaluation struct {
	QuestionIndex int    `json:"questionIndex"`
	Question      string `json:"question"`
	Category      string `json:"category"`
	UserAnswer    string `json:"userAnswer"`
	Score         int    `json:"score"`
	Feedback      string `json:"feedback"`
}

type ReferenceAnswer struct {
	QuestionIndex   int      `json:"questionIndex"`
	Question        string   `json:"question"`
	ReferenceAnswer string   `json:"referenceAnswer"`
	KeyPoints       []string `json:"keyPoints"`
}

// SessionReportDB 从 interview_sessions 表装载的报告所需列。
type SessionReportDB struct {
	InternalID           int64
	SessionID            string
	Status               string
	TotalQuestions       *int
	QuestionsJSON        string
	OverallScore         *int
	OverallFeedback      string
	StrengthsJSON        string
	ImprovementsJSON     string
	ReferenceAnswersJSON string
	EvaluateStatus       string
	EvaluateError        string
	CreatedAt            time.Time
	CompletedAt          *time.Time
}

// InterviewAnswerDB 从 interview_answers 行映射，供与 questions_json 合并。
type InterviewAnswerDB struct {
	QuestionIndex   int
	Question        string
	Category        string
	UserAnswer      string
	Score           *int
	Feedback        string
	ReferenceAnswer string
	KeyPointsJSON   string
	AnsweredAt      time.Time
}
