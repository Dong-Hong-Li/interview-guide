package dto

// 以下为 LLM 面试评估与 dto.InterviewReport 契约所需子集（与主项目 DTO 字段对齐，避免拉入 persistence 依赖）。

// InterviewQuestion 评估时输入的问答项。
type InterviewQuestion struct {
	QuestionIndex       int    `json:"questionIndex"`
	Question            string `json:"question"`
	Type                string `json:"type"`
	Category            string `json:"category"`
	UserAnswer          string `json:"userAnswer,omitempty"`
	Score               *int   `json:"score,omitempty"`
	Feedback            string `json:"feedback,omitempty"`
	IsFollowUp          bool   `json:"isFollowUp"`
	ParentQuestionIndex *int   `json:"parentQuestionIndex,omitempty"`
}

// InterviewReport LLM 汇总后的整卷报告（与 Java AnswerEvaluationService 对齐的对外子集）。
type InterviewReport struct {
	SessionID        string                     `json:"sessionId"`
	TotalQuestions   int                        `json:"totalQuestions"`
	OverallScore     int                        `json:"overallScore"`
	CategoryScores   []InterviewCategoryScore   `json:"categoryScores"`
	QuestionDetails  []InterviewQuestionEval    `json:"questionDetails"`
	OverallFeedback  string                     `json:"overallFeedback"`
	Strengths        []string                   `json:"strengths"`
	Improvements     []string                   `json:"improvements"`
	ReferenceAnswers []InterviewReferenceAnswer `json:"referenceAnswers"`
}

// InterviewCategoryScore 分类得分汇总。
type InterviewCategoryScore struct {
	Category      string `json:"category"`
	Score         int    `json:"score"`
	QuestionCount int    `json:"questionCount"`
}

// InterviewQuestionEval 逐题评估。
type InterviewQuestionEval struct {
	QuestionIndex int    `json:"questionIndex"`
	Question      string `json:"question"`
	Category      string `json:"category"`
	UserAnswer    string `json:"userAnswer"`
	Score         int    `json:"score"`
	Feedback      string `json:"feedback"`
}

// InterviewReferenceAnswer 单题参考要点（用于落库 reference_answers_json）。
type InterviewReferenceAnswer struct {
	QuestionIndex   int      `json:"questionIndex"`
	Question        string   `json:"question"`
	ReferenceAnswer string   `json:"referenceAnswer"`
	KeyPoints       []string `json:"keyPoints"`
}
