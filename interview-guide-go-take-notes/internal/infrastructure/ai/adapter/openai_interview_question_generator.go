package adapter

import (
	"context"

	res "interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"

	"go.uber.org/zap"
)

// OpenAIInterviewQuestionGenerator 包装 *ai.InterviewQuestionGenerator（见 interview_questions.go），实现 repository.InterviewQuestionGenerator。
type OpenAIInterviewQuestionGenerator struct {
	core *ai.InterviewQuestionGenerator
}

// NewOpenAIInterviewQuestionGenerator 使用与简历 AI 相同的模型/截断/温度；followUpCount 取 1。
func NewOpenAIInterviewQuestionGenerator(oa *ai.OpenAIService, cfg *config.Config, lg *zap.Logger) *OpenAIInterviewQuestionGenerator {
	if oa == nil {
		return nil
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	aiCfg := config.OpenAIConfig{}
	if cfg != nil {
		aiCfg = cfg.Openai
	}
	mr := aiCfg.ResumeAIMaxRunes
	if mr <= 0 {
		mr = 120_000
	}
	mct := aiCfg.ResumeAIMaxCompletionTokens
	if mct < 2048 {
		mct = 8192
	}
	core := ai.NewInterviewQuestionGenerator(
		oa.Client(),
		aiCfg.AIModel,
		mr,
		mct,
		aiCfg.ResumeAITemperature,
		1,
		lg,
	)
	return &OpenAIInterviewQuestionGenerator{core: core}
}

// GenerateQuestions 委托 GenerateForQueue（LLM 失败则返回 error，不静默降级）。
func (g *OpenAIInterviewQuestionGenerator) GenerateQuestions(ctx context.Context, resumeText string, questionCount int, historical []string, interviewerRole string) ([]res.InterviewQuestion, error) {
	if g == nil || g.core == nil {
		return nil, nil
	}
	return g.core.GenerateForQueue(ctx, resumeText, questionCount, historical, interviewerRole)
}

var _ repository.InterviewQuestionGenerator = (*OpenAIInterviewQuestionGenerator)(nil)
