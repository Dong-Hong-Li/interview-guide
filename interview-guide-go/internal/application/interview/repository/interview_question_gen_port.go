package repository

import (
	"context"

	"interview-guide-go/internal/application/interview/model/results"
)

// InterviewQuestionGenerator 面试题目生成器（与 `internal/infrastructure/ai/adapter/interview_questions.go` 行为对齐：模板、分布、追问展开）。
type InterviewQuestionGenerator interface {
	GenerateQuestions(ctx context.Context, resumeText string, questionCount int, historicalQuestions []string, interviewerRole string) ([]results.InterviewQuestion, error)
}
