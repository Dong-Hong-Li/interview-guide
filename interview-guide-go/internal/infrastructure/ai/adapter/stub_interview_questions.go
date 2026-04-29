package adapter

import (
	"context"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"

	"go.uber.org/zap"
)

// StubInterviewQuestionGenerator 在 OpenAI 未就绪时注入，复用 *InterviewQuestionGenerator 的内置默认题库返回兜底题。
type StubInterviewQuestionGenerator struct {
	core *InterviewQuestionGenerator
}

func NewStubInterviewQuestionGenerator() *StubInterviewQuestionGenerator {
	return &StubInterviewQuestionGenerator{core: &InterviewQuestionGenerator{followUpCount: 0, lg: zap.NewNop()}}
}

// GenerateQuestions 将请求交给内置 defaultQuestions，按 interviewerRole 选取后端/前端默认题集。
func (s *StubInterviewQuestionGenerator) GenerateQuestions(_ context.Context, _ string, questionCount int, _ []string, interviewerRole string) ([]results.InterviewQuestion, error) {
	if s == nil || s.core == nil {
		return nil, nil
	}
	if questionCount < 1 {
		questionCount = 1
	}
	if questionCount > 30 {
		questionCount = 30
	}
	return s.core.defaultQuestions(questionCount, interviewerRole), nil
}

var _ repository.InterviewQuestionGenerator = (*StubInterviewQuestionGenerator)(nil)
