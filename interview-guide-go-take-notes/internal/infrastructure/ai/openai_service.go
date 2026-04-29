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
		return nil, errors.New(errmsg.ConfigOpenAIServiceStartFailed)
	}

	baseURL := strings.TrimSpace(c.Openai.OpenAIBaseURL)
	oaClient := openaisdk.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &OpenAIService{client: oaClient}, nil
}

// EmbeddingHTTPClient 返回知识库向量专用 HTTP 客户端（与聊天 Moonshot/OpenAI 分离）。
// GatewayAPIKey / GatewayBaseURL 须在配置阶段校验为非空（禁止静默默认网关 URL）。
func EmbeddingHTTPClient(cfg *config.Config, _ *OpenAIService) (openaisdk.Client, error) {
	if cfg == nil {
		return openaisdk.Client{}, errors.New(errmsg.ConfigOpenAIServiceStartFailed)
	}
	key := strings.TrimSpace(cfg.Openai.KnowledgeEmbedding.GatewayAPIKey)
	if key == "" {
		return openaisdk.Client{}, errors.New(errmsg.ConfigKnowledgeEmbeddingGatewayAPIKeyRequired)
	}
	base := strings.TrimSpace(cfg.Openai.KnowledgeEmbedding.GatewayBaseURL)
	if base == "" {
		return openaisdk.Client{}, errors.New(errmsg.ConfigKnowledgeEmbeddingGatewayBaseURLRequired)
	}
	return openaisdk.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(base),
	), nil
}
