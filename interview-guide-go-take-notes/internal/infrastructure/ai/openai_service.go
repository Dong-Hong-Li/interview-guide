package ai

import (
	"context"
	"errors"
	"strings"

	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIService 封装官方 OpenAI 兼容 HTTP 客户端（Kimi/GLM/OpenAI 等经 BaseURL 切换），供简历打分、面试出题/评估等复用。
type OpenAIService struct {
	client openaisdk.Client
}

// Client 返回底层 SDK 客户端，供 InterviewQuestionGenerator、ResumeGrader、InterviewEvaluator 等注入。
func (s *OpenAIService) Client() openaisdk.Client {
	if s == nil {
		return openaisdk.Client{}
	}
	return s.client
}

// NewOpenAIService 从配置创建 OpenAI 兼容客户端；配置未就绪时返回错误（与 cmd 启动逻辑一致）。
func NewOpenAIService(_ context.Context, c *config.Config) (*OpenAIService, error) {
	if c == nil || !c.Openai.OpenAIConfigured() {
		return nil, errors.New(errmsg.ConfigOpenAIServiceStartFailed)
	}
	apiKey := strings.TrimSpace(c.Openai.OpenAIAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.Openai.MoonshotAPIKey)
	}
	if apiKey == "" {
		return nil, errors.New(errmsg.ConfigOpenAIServiceStartFailed)
	}

	baseURL := strings.TrimSpace(c.Openai.OpenAIBaseURL)
	oaClient := openaisdk.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &OpenAIService{client: oaClient}, nil
}
