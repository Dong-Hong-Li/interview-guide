// Package adapter 中的本文件实现「上传文档全文 → 语义分块」：在向量化写入 PostgreSQL 之前，
// 需先将抽取的正文切成多块「语义连贯」的字符串；过长整块 embedding 效果差，且超过模型上下文则需切块。
//
// 调用方：Redis Stream 消费者 KnowledgeVectorizeConsumer；输入为对象存储上的全文或抽取文本，输出 chunks[] 再逐块 Embedding。
// 网关：使用主 OPENAI_* Chat Completions（与 Embeddings 网关分离），便于用较强模型控制 JSON 输出格式。
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/internal/config"
	"interview-guide-go/internal/infrastructure/ai"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/logmsg"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	constpkg "github.com/openai/openai-go/shared/constant"
	"go.uber.org/zap"
)

// knowledgeChunkSystemPrompt 约束模型仅输出 JSON（chunks + exceptions），避免正文混入 fenced code 块导致解析失败。
//
// 业务上：exceptions 中的片段不入库向量化，仅记录审计（乱码/损坏字节），避免污染检索结果。
const knowledgeChunkSystemPrompt = `你是知识库文本分片助手。用户会提供一篇从 PDF 等文件抽取的全文，你需要将其切分为适合向量检索与 RAG 的若干文本块。

硬性规则：
1) 输出必须是**一个 JSON 对象**，且仅包含两个键：chunks、exceptions。不要输出 Markdown 代码围栏、不要输出任何解释性前后文。
2) chunks 为字符串数组：每一段应是连续、可独立检索的正文；尽量在段落或小节边界处切分；单段建议不超过约 1500 个 Unicode 字符（略超可接受，但不要过长）。
3) 尽量保留原文表述，不要改写、不要摘要、不要翻译；不要编造正文中不存在的内容。
4) 若你识别到**乱码**、**明显不可读的替换符/控制字符成串**、**疑似编码损坏**、或**与上下文完全无法拼接的字节碎片**，**禁止**放进 chunks；必须放入 exceptions。
5) exceptions 为对象数组，每项含两个字符串键：raw_excerpt（摘录问题片段，建议不超过 200 个字符）、reason（简短说明，例如「疑似 UTF-8 乱码」「PDF 抽取噪声」）。
6) 若全文正常、无上述问题，exceptions 应为空数组 []。
7) chunks 中每个字符串不要为纯空白；不要输出空字符串元素。`

// OpenAIKnowledgeTextChunker 使用 Chat Completions + JSON object 响应格式，将全文拆成 KnowledgeChunkSplitResult。
//
// 字段含义见端口 KnowledgeTextChunker：Chunks 进入后续 Embed；Exceptions 仅供日志与排障。
type OpenAIKnowledgeTextChunker struct {
	client openai.Client
	model  shared.ChatModel
	// openaiBaseURL 来自 OPENAI_BASE_URL，仅日志核对「请求发到哪」，不参与路由逻辑。
	openaiBaseURL string
	// maxInputRunes 防止超大文档撑爆上下文：超长时按 rune 截断并打日志（牺牲尾部内容换取任务可完成）。
	maxInputRunes int
	// maxCompletionTokens 限制模型输出 JSON 长度；过小可能导致 chunks 被截断致解析失败。
	maxCompletionTokens int64
	temperature         float64
	lg                  *zap.Logger
}

var _ kbrepo.KnowledgeTextChunker = (*OpenAIKnowledgeTextChunker)(nil)

// NewOpenAIKnowledgeTextChunker 由 deps 注入消费者；KnowledgeChunkingConfig 未填时使用合理默认（模型回落到全局 AIModel 或 gpt-4o-mini）。
func NewOpenAIKnowledgeTextChunker(client openai.Client, cfg *config.Config, lg *zap.Logger) kbrepo.KnowledgeTextChunker {
	if lg == nil {
		lg = zap.NewNop()
	}
	var c config.KnowledgeChunkingConfig
	if cfg != nil {
		c = cfg.Openai.KnowledgeChunking
	}
	if c.MaxInputRunes <= 0 {
		c.MaxInputRunes = 200000
	}
	if c.MaxCompletionTokens <= 0 {
		c.MaxCompletionTokens = 32768
	}
	model := strings.TrimSpace(c.Model)
	if model == "" && cfg != nil {
		model = strings.TrimSpace(cfg.Openai.AIModel)
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	baseURL := ""
	if cfg != nil {
		baseURL = strings.TrimSpace(cfg.Openai.OpenAIBaseURL)
	}
	return &OpenAIKnowledgeTextChunker{
		client:              client,
		model:               shared.ChatModel(model),
		openaiBaseURL:       baseURL,
		maxInputRunes:       c.MaxInputRunes,
		maxCompletionTokens: c.MaxCompletionTokens,
		temperature:         c.Temperature,
		lg:                  lg.Named("knowledge_chunk_ai"),
	}
}

// SplitForVectorize 实现端口 KnowledgeTextChunker：对单篇全文输出结构化分块结果。
//
// 流程概要：校验非空 → 超长截断 → 调 Chat JSON → 从模型原文中用 ExtractJSONObject 抽出 JSON → 反序列化为领域约定结构。
// Context 若注入 kbId，则日志贯穿「正在处理哪条知识库」，便于并行消费者排查。
func (c *OpenAIKnowledgeTextChunker) SplitForVectorize(ctx context.Context, fullText string) (kbrepo.KnowledgeChunkSplitResult, error) {
	var empty kbrepo.KnowledgeChunkSplitResult
	if c == nil {
		return empty, fmt.Errorf("%sclient nil", errmsg.KnowledgeBaseChunkAIFailedPrefix)
	}
	text := strings.TrimSpace(fullText)
	if text == "" {
		return empty, fmt.Errorf("%sempty text", errmsg.KnowledgeBaseChunkAIFailedPrefix)
	}
	// 超长截断：否则 prompt token 爆炸或网关拒收；业务上建议前端提示「超大文档分段上传」，此处仅兜底。
	if n := utf8.RuneCountInString(text); n > c.maxInputRunes {
		c.lg.Info(logmsg.MsgKnowledgeChunkInputTruncated,
			zap.Int(logmsg.FieldKnowledgeChunkInputRunes, n),
			zap.Int(logmsg.FieldMaxRunes, c.maxInputRunes))
		text = string([]rune(text)[:c.maxInputRunes])
	}

	userBody := "以下是需要分片的原文（请严格按系统说明仅输出 JSON）：\n\n" + text

	// ResponseFormat 强制 JSON object，与 PtrJSONObjectFormat 配套；旧模型不支持 schema 时仍能得到 `{}` 形状输出。
	params := openai.ChatCompletionNewParams{
		Model:               c.model,
		MaxCompletionTokens: openai.Int(c.maxCompletionTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(knowledgeChunkSystemPrompt),
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Role: constpkg.User("user"),
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(userBody),
					},
				},
			},
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: ai.PtrJSONObjectFormat(),
		},
	}
	if c.temperature > 0 {
		params.Temperature = openai.Float(c.temperature)
	}

	inputRunes := utf8.RuneCountInString(text)
	beginFields := []zap.Field{
		zap.String(logmsg.FieldOpenAIBaseURL, c.openaiBaseURL),
		zap.String(logmsg.FieldModel, string(c.model)),
		zap.Int(logmsg.FieldKnowledgeChunkInputRunes, inputRunes),
		zap.Int64("maxCompletionTokens", c.maxCompletionTokens),
	}
	beginFields = appendKbIDFields(ctx, beginFields)
	c.lg.Info(logmsg.MsgKnowledgeChunkAIBegin, beginFields...)

	llmStart := time.Now()
	resp, err := c.client.Chat.Completions.New(ctx, params)
	llmDur := time.Since(llmStart)
	if err != nil {
		failFields := []zap.Field{
			zap.String(logmsg.FieldOpenAIBaseURL, c.openaiBaseURL),
			zap.Duration(logmsg.FieldLLMDuration, llmDur),
			zap.String(logmsg.FieldModel, string(c.model)),
			zap.Error(err),
		}
		failFields = appendKbIDFields(ctx, failFields)
		c.lg.Warn(logmsg.MsgKnowledgeChunkAIInvokeFailed, failFields...)
		return empty, fmt.Errorf("%s%w", errmsg.KnowledgeBaseChunkAIFailedPrefix, err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		failFields := []zap.Field{
			zap.String(logmsg.FieldOpenAIBaseURL, c.openaiBaseURL),
			zap.Duration(logmsg.FieldLLMDuration, llmDur),
			zap.String(logmsg.FieldModel, string(c.model)),
			zap.String(logmsg.FieldReason, "no_completion_choices"),
		}
		failFields = appendKbIDFields(ctx, failFields)
		c.lg.Warn(logmsg.MsgKnowledgeChunkAIInvokeFailed, failFields...)
		return empty, fmt.Errorf("%sno completion choices", errmsg.KnowledgeBaseChunkAIFailedPrefix)
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	// 模型偶发输出 Markdown 或前缀废话：抽取首个 `{...}` 再解析，提高成功率。
	raw = ai.ExtractJSONObject(raw)
	out, decErr := decodeKnowledgeChunkResponse(raw)
	if decErr != nil {
		parseFields := []zap.Field{
			zap.String(logmsg.FieldOpenAIBaseURL, c.openaiBaseURL),
			zap.Duration(logmsg.FieldLLMDuration, llmDur),
			zap.String(logmsg.FieldModel, string(c.model)),
			zap.String("completionId", resp.ID),
			zap.Error(decErr),
			zap.String("rawPreview", truncateForLog(raw, 500)),
		}
		parseFields = appendKbIDFields(ctx, parseFields)
		c.lg.Warn(logmsg.MsgKnowledgeChunkAIParseFailed, parseFields...)
		return empty, decErr
	}

	okFields := []zap.Field{
		zap.String(logmsg.FieldOpenAIBaseURL, c.openaiBaseURL),
		zap.Duration(logmsg.FieldLLMDuration, llmDur),
		zap.String(logmsg.FieldModel, string(c.model)),
		zap.Int("chunkCount", len(out.Chunks)),
		zap.Int("exceptionCount", len(out.Exceptions)),
		zap.String("finishReason", resp.Choices[0].FinishReason),
		zap.String("completionId", resp.ID),
		zap.String("responseModel", resp.Model),
		zap.Int("responseContentRunes", utf8.RuneCountInString(strings.TrimSpace(resp.Choices[0].Message.Content))),
	}
	if resp.JSON.Usage.Valid() {
		okFields = append(okFields,
			zap.Int64("promptTokens", resp.Usage.PromptTokens),
			zap.Int64("completionTokens", resp.Usage.CompletionTokens),
			zap.Int64("totalTokens", resp.Usage.TotalTokens),
		)
	}
	okFields = appendKbIDFields(ctx, okFields)
	c.lg.Info(logmsg.MsgKnowledgeChunkAIInvokeOK, okFields...)

	return out, nil
}

// appendKbIDFields 在向量化流水线日志中附加 kbId（若 Context 已通过 KnowledgeBaseVectorizeID 注入）。
func appendKbIDFields(ctx context.Context, fields []zap.Field) []zap.Field {
	if kbID, ok := kbrepo.KnowledgeBaseVectorizeIDFromContext(ctx); ok {
		return append(fields, zap.Int64("kbId", kbID))
	}
	return fields
}

// truncateForLog 截断过长模型输出，避免日志刷屏；不影响业务，仅用于 rawPreview。
func truncateForLog(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// rawKnowledgeChunkResponse 与 LLM 输出的 JSON 字段名对齐（snake_case），再转换为端口类型 KnowledgeChunkSplitResult。
type rawKnowledgeChunkResponse struct {
	Chunks     []string `json:"chunks"`
	Exceptions []struct {
		RawExcerpt string `json:"raw_excerpt"`
		Reason     string `json:"reason"`
	} `json:"exceptions"`
}

// decodeKnowledgeChunkResponse 将模型 JSON 映射为仓储层结构：空白 chunk 丢弃；exceptions 双空则跳过。
func decodeKnowledgeChunkResponse(raw string) (kbrepo.KnowledgeChunkSplitResult, error) {
	var empty kbrepo.KnowledgeChunkSplitResult
	var parsed rawKnowledgeChunkResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return empty, fmt.Errorf("%sparse model json: %w", errmsg.KnowledgeBaseChunkAIFailedPrefix, err)
	}
	out := kbrepo.KnowledgeChunkSplitResult{
		Chunks:     make([]string, 0, len(parsed.Chunks)),
		Exceptions: make([]kbrepo.KnowledgeChunkException, 0, len(parsed.Exceptions)),
	}
	for _, ch := range parsed.Chunks {
		t := strings.TrimSpace(ch)
		if t != "" {
			out.Chunks = append(out.Chunks, t)
		}
	}
	for _, ex := range parsed.Exceptions {
		reason := strings.TrimSpace(ex.Reason)
		samp := strings.TrimSpace(ex.RawExcerpt)
		if reason == "" && samp == "" {
			continue
		}
		out.Exceptions = append(out.Exceptions, kbrepo.KnowledgeChunkException{
			RawExcerpt: samp,
			Reason:     reason,
		})
	}
	return out, nil
}
