package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"interview-guide-go/shared/errmsg"
)

// ─────────────────────────────────────────────────────────────────────────────
// 知识库向量（Embedding）专用环境变量名（仅文档与 Parse 共用，勿散落魔法字符串）。
// 与 OPENAI_BASE_URL / Moonshot 聊天分离：Moonshot 不提供 OpenAI 兼容的 POST /v1/embeddings。
// ─────────────────────────────────────────────────────────────────────────────
const (
	// EnvKnowledgeEmbeddingGatewayAPIKey 向量网关 API Key。
	EnvKnowledgeEmbeddingGatewayAPIKey = "KB_EMBEDDING_OPENAI_API_KEY"
	// EnvKnowledgeEmbeddingGatewayBaseURL 向量网关 Base URL（须兼容 OpenAI Embeddings 路径）；必填。
	EnvKnowledgeEmbeddingGatewayBaseURL = "KB_EMBEDDING_OPENAI_BASE_URL"
	// EnvKnowledgeEmbeddingModel POST /embeddings 所用模型名；必填。
	EnvKnowledgeEmbeddingModel = "KB_EMBEDDING_MODEL"
	// EnvKnowledgeEmbeddingDimensions PG `vector(N)` 列维度；必填，须与迁移脚本一致。
	EnvKnowledgeEmbeddingDimensions = "KB_EMBEDDING_DIMENSIONS"
	// EnvKnowledgeEmbeddingBatchSize 单次 POST /embeddings 的 input 条数上限；留空默认 10（阿里云 DashScope 要求≤10）。
	EnvKnowledgeEmbeddingBatchSize = "KB_EMBEDDING_BATCH_SIZE"

	// EnvKnowledgeChunkAIModel 知识库 AI 分片所用 Chat 模型；留空则使用 AI_MODEL。
	EnvKnowledgeChunkAIModel = "KB_CHUNK_AI_MODEL"
	// EnvKnowledgeChunkMaxInputRunes 分片请求正文最大 rune 数；留空则使用 RESUME_AI_MAX_RUNES。
	EnvKnowledgeChunkMaxInputRunes = "KB_CHUNK_MAX_INPUT_RUNES"
	// EnvKnowledgeChunkMaxCompletionTokens 分片模型最大输出 token；留空默认 32768。上限对齐 Moonshot kimi-k2.5 等 256K 总窗口（须满足 prompt_tokens+max_completion≤262144）。
	EnvKnowledgeChunkMaxCompletionTokens = "KB_CHUNK_MAX_COMPLETION_TOKENS"
	// EnvKnowledgeChunkTemperature 分片采样温度；留空则与 RESUME_AI_TEMPERATURE 一致（Moonshot 等模型常要求为 1）；可设为 0 以省略请求体中的 temperature 字段。
	EnvKnowledgeChunkTemperature = "KB_CHUNK_TEMPERATURE"
)

const (
	defaultKnowledgeEmbeddingBatchSize = 10
	maxKnowledgeEmbeddingBatchSize     = 2048

	defaultKnowledgeChunkMaxCompletionTokens int64 = 32768
	minKnowledgeChunkMaxCompletionTokens     int64 = 1024
	// maxKnowledgeChunkMaxCompletionTokens Moonshot kimi-k2.5 文档：上下文 256K tokens；单次请求须 prompt_tokens+max_completion_tokens≤262144。
	maxKnowledgeChunkMaxCompletionTokens int64 = 256 * 1024 // 262144
	minKnowledgeChunkMaxInputRunes             = 1024
)

// KnowledgeEmbeddingConfig 知识库异步向量化专用：网关 + 模型与维度。
type KnowledgeEmbeddingConfig struct {
	GatewayAPIKey  string
	GatewayBaseURL string
	Model          string
	Dimensions     int
	// BatchSize 单次 Embeddings 请求的文本条数；默认 10 以兼容 DashScope，OpenAI 等可调大（见 KB_EMBEDDING_BATCH_SIZE）。
	BatchSize int
}

// KnowledgeChunkingConfig 知识库向量化前全文分片：走 OPENAI_BASE_URL + Chat Completions（与 Embeddings 网关分离）。
// Temperature 为 0 时请求体不传 temperature（与 ResumeGrader 一致）；非 0 时原样上传（Moonshot 部分模型仅允许 1，请与 RESUME_AI_TEMPERATURE 或 KB_CHUNK_TEMPERATURE 对齐）。
type KnowledgeChunkingConfig struct {
	Model               string
	MaxInputRunes       int
	MaxCompletionTokens int64
	Temperature         float64
}

type OpenAIConfig struct {
	OpenAIAPIKey                string
	OpenAIBaseURL               string
	MoonshotAPIKey              string
	AIModel                     string
	ResumeAIMaxRunes            int
	ResumeAIMaxCompletionTokens int64
	ResumeAITemperature         float64
	KnowledgeEmbedding          KnowledgeEmbeddingConfig
	KnowledgeChunking           KnowledgeChunkingConfig
}

// validateOpenAIConfig 校验 OpenAI / Moonshot / 简历 AI / 知识库 Embedding 相关配置（缺失或无效即报错）。
func validateOpenAIConfig() (*OpenAIConfig, error) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		return nil, errors.New(errmsg.ConfigOpenAIAPIKeyRequired)
	}
	openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseURL == "" {
		return nil, errors.New(errmsg.ConfigOpenAIBaseURLRequired)
	}
	moonshotAPIKey := os.Getenv("MOONSHOT_API_KEY")
	if moonshotAPIKey == "" {
		return nil, errors.New(errmsg.ConfigMoonshotAPIKeyRequired)
	}
	aiModel := os.Getenv("AI_MODEL")
	if aiModel == "" {
		return nil, errors.New(errmsg.ConfigAIModelRequired)
	}

	resumeAIMaxRunes, err := requirePositiveInt(
		"RESUME_AI_MAX_RUNES",
		errmsg.ConfigResumeAIMaxRunesRequired,
		errmsg.ConfigResumeAIMaxRunesInvalid,
	)
	if err != nil {
		return nil, err
	}
	resumeAIMaxCompletionTokens, err := requirePositiveInt64(
		"RESUME_AI_MAX_COMPLETION_TOKENS",
		errmsg.ConfigResumeAIMaxCompletionTokensRequired,
		errmsg.ConfigResumeAIMaxTokensInvalid,
	)
	if err != nil {
		return nil, err
	}

	resumeAITemperature, err := strconv.ParseFloat(os.Getenv("RESUME_AI_TEMPERATURE"), 64)
	if err != nil || resumeAITemperature < 0 {
		return nil, errors.New(errmsg.ConfigResumeAITemperatureInvalid)
	}

	kbEmbedModel := strings.TrimSpace(os.Getenv(EnvKnowledgeEmbeddingModel))
	if kbEmbedModel == "" {
		return nil, errors.New(errmsg.ConfigKnowledgeEmbeddingModelRequired)
	}

	rawDims := strings.TrimSpace(os.Getenv(EnvKnowledgeEmbeddingDimensions))
	if rawDims == "" {
		return nil, errors.New(errmsg.ConfigKnowledgeBaseEmbeddingDimensionsRequired)
	}
	kbEmbedDims, err := strconv.Atoi(rawDims)
	if err != nil || kbEmbedDims < 256 || kbEmbedDims > 8192 {
		return nil, errors.New(errmsg.ConfigKnowledgeBaseEmbeddingDimensionsInvalid)
	}

	kbEmbAPIKey := strings.TrimSpace(os.Getenv(EnvKnowledgeEmbeddingGatewayAPIKey))
	if kbEmbAPIKey == "" {
		return nil, errors.New(errmsg.ConfigKnowledgeEmbeddingGatewayAPIKeyRequired)
	}
	kbEmbBaseURL := strings.TrimSpace(os.Getenv(EnvKnowledgeEmbeddingGatewayBaseURL))
	if kbEmbBaseURL == "" {
		return nil, errors.New(errmsg.ConfigKnowledgeEmbeddingGatewayBaseURLRequired)
	}

	kbEmbedBatch := defaultKnowledgeEmbeddingBatchSize
	if s := strings.TrimSpace(os.Getenv(EnvKnowledgeEmbeddingBatchSize)); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > maxKnowledgeEmbeddingBatchSize {
			return nil, errors.New(errmsg.ConfigKnowledgeEmbeddingBatchSizeInvalid)
		}
		kbEmbedBatch = n
	}

	kbChunkModel := aiModel
	if m := strings.TrimSpace(os.Getenv(EnvKnowledgeChunkAIModel)); m != "" {
		kbChunkModel = m
	}

	kbChunkMaxIn := resumeAIMaxRunes
	if s := strings.TrimSpace(os.Getenv(EnvKnowledgeChunkMaxInputRunes)); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < minKnowledgeChunkMaxInputRunes {
			return nil, errors.New(errmsg.ConfigKnowledgeChunkMaxInputRunesInvalid)
		}
		kbChunkMaxIn = n
	}

	kbChunkMaxOut := defaultKnowledgeChunkMaxCompletionTokens
	if s := strings.TrimSpace(os.Getenv(EnvKnowledgeChunkMaxCompletionTokens)); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < minKnowledgeChunkMaxCompletionTokens || n > maxKnowledgeChunkMaxCompletionTokens {
			return nil, errors.New(errmsg.ConfigKnowledgeChunkMaxCompletionTokensInvalid)
		}
		kbChunkMaxOut = n
	}

	// 与简历/面试共用 RESUME_AI_TEMPERATURE，避免 Moonshot 等网关报「only 1 is allowed」而分片仍发默认 0.2。
	kbChunkTemp := resumeAITemperature
	if s := strings.TrimSpace(os.Getenv(EnvKnowledgeChunkTemperature)); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v < 0 || v > 2 {
			return nil, errors.New(errmsg.ConfigKnowledgeChunkTemperatureInvalid)
		}
		kbChunkTemp = v
	}

	return &OpenAIConfig{
		OpenAIAPIKey:                openaiAPIKey,
		OpenAIBaseURL:               openaiBaseURL,
		MoonshotAPIKey:              moonshotAPIKey,
		AIModel:                     aiModel,
		ResumeAIMaxRunes:            resumeAIMaxRunes,
		ResumeAIMaxCompletionTokens: resumeAIMaxCompletionTokens,
		ResumeAITemperature:         resumeAITemperature,
		KnowledgeEmbedding: KnowledgeEmbeddingConfig{
			GatewayAPIKey:  kbEmbAPIKey,
			GatewayBaseURL: kbEmbBaseURL,
			Model:          kbEmbedModel,
			Dimensions:     kbEmbedDims,
			BatchSize:      kbEmbedBatch,
		},
		KnowledgeChunking: KnowledgeChunkingConfig{
			Model:               kbChunkModel,
			MaxInputRunes:       kbChunkMaxIn,
			MaxCompletionTokens: kbChunkMaxOut,
			Temperature:         kbChunkTemp,
		},
	}, nil
}

// OpenAIConfigured 聊天侧 OpenAI 兼容客户端是否可用（已通过 validateOpenAIConfig 时恒为 true）。
func (c *OpenAIConfig) OpenAIConfigured() bool {
	return c != nil && strings.TrimSpace(c.OpenAIAPIKey) != ""
}

func requirePositiveInt(name, missingMsg, invalidMsg string) (int, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return 0, errors.New(missingMsg)
	}
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return 0, errors.New(invalidMsg)
	}
	return i, nil
}

func requirePositiveInt64(name, missingMsg, invalidMsg string) (int64, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return 0, errors.New(missingMsg)
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil || i <= 0 {
		return 0, errors.New(invalidMsg)
	}
	return i, nil
}
