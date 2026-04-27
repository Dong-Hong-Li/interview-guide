package model

// EvaluationReport 异步评估落库用报告（与 LLM 及 SaveInterviewEvaluationResult 对齐）。
type EvaluationReport struct {
	OverallScore     int
	OverallFeedback  string
	Strengths        []string
	Improvements     []string
	QuestionDetails  []EvaluationQuestionDetail
	ReferenceAnswers []EvaluationReferenceAnswer
}

// EvaluationQuestionDetail 逐题评估结果。
type EvaluationQuestionDetail struct {
	QuestionIndex int
	Question      string
	Category      string
	UserAnswer    string
	Score         int
	Feedback      string
}

// EvaluationReferenceAnswer 参考答案与要点。
type EvaluationReferenceAnswer struct {
	QuestionIndex   int
	Question        string
	ReferenceAnswer string
	KeyPoints       []string
}
