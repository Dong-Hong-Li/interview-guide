// Package adapter 中的本文件实现知识库「问答生成」侧的 Chat Completions 调用。
//
// 业务链路：用户提问 → 服务层将检索到的若干文本块与问题拼成 prompt → 由本适配器调用主 OPENAI_* 网关
// 生成自然语言答案（非流式或流式 SSE）。与 Embeddings（KB_EMBEDDING_*）无关：前者是生成任务，后者是编码任务。
package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/internal/infrastructure/ai"

	"github.com/openai/openai-go"
	constpkg "github.com/openai/openai-go/shared/constant"
)

// KnowledgeBaseQueryChatAdapter 实现端口 KnowledgeBaseQueryChat：仅负责「system + user 两段消息 → 模型输出」，
// 不负责检索与 prompt 拼装（由 application 层 KnowledgeBaseQueryService 完成）。
//
// 与简历评分、知识库分片等共用 OpenAIService 封装的同一个 Chat 客户端（BaseURL/API Key），配置上的模型名、温度、最大输出长度来自业务约定。
type KnowledgeBaseQueryChatAdapter struct {
	client openai.Client
	model  openai.ChatModel
	maxTok int64
	temp   float64
}

// NewKnowledgeBaseQueryChatAdapter 由 Wire 的 provideKnowledgeBaseQueryChat 注入 RAG 查询服务。
//
// 参数摘自 config.Openai：AIModel、ResumeAIMaxCompletionTokens、ResumeAITemperature 等与「主对话」一致，
// 保证知识库问答与站内其它 LLM 能力在运维上可统一调参（若需单独调参可在配置中拆分字段后再改此处）。
func NewKnowledgeBaseQueryChatAdapter(oa *ai.OpenAIService, model string, maxTok int64, temp float64) *KnowledgeBaseQueryChatAdapter {
	return &KnowledgeBaseQueryChatAdapter{
		client: oa.Client(),
		model:  openai.ChatModel(strings.TrimSpace(model)),
		maxTok: maxTok,
		temp:   temp,
	}
}

var _ kbrepo.KnowledgeBaseQueryChat = (*KnowledgeBaseQueryChatAdapter)(nil)

// Complete 用于 POST /query 等非流式场景：一次性返回整段助手回复，便于 JSON 封装给前端。
func (a *KnowledgeBaseQueryChatAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if a == nil {
		return "", errors.New("nil KnowledgeBaseQueryChatAdapter")
	}
	params := openai.ChatCompletionNewParams{
		Model: a.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(systemPrompt),
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Role: constpkg.User("user"),
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(userPrompt),
					},
				},
			},
		},
		MaxCompletionTokens: openai.Int(a.maxTok),
	}
	if a.temp > 0 {
		params.Temperature = openai.Float(a.temp)
	}
	resp, err := a.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// Stream 用于 POST /query/stream、RAG 会话 messages/stream：按模型增量回调 onDelta，上层负责写入 SSE 或累加全文。
//
// 若上游取消 ctx（用户断开），底层 stream 会结束并返回 ctx 错误，业务层需停止写入 HTTP 响应。
func (a *KnowledgeBaseQueryChatAdapter) Stream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(fragment string) error) error {
	if a == nil {
		return errors.New("nil KnowledgeBaseQueryChatAdapter")
	}
	if onDelta == nil {
		onDelta = func(string) error { return nil }
	}
	params := openai.ChatCompletionNewParams{
		Model: a.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(systemPrompt),
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Role: constpkg.User("user"),
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(userPrompt),
					},
				},
			},
		},
		MaxCompletionTokens: openai.Int(a.maxTok),
	}
	if a.temp > 0 {
		params.Temperature = openai.Float(a.temp)
	}
	stream := a.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()
	for stream.Next() {
		ch := stream.Current()
		for _, c := range ch.Choices {
			t := c.Delta.Content
			if t == "" {
				continue
			}
			if err := onDelta(t); err != nil {
				return err
			}
		}
	}
	return stream.Err()
}
