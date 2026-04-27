package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"interview-guide-go/shared/errmsg"
)

// 与 200K token 能力常见的控制台配置对齐：简历侧与输出侧默认拉满到 200K 档位（实际仍受模型总上下文与计费限制）。
const (
	defaultResumeAIMaxRunes            = 200_000
	defaultResumeAIMaxCompletionTokens = 200_000
)

type OpenAIConfig struct {
	// openai api 密钥
	OpenAIAPIKey string
	// openai 基础 URL
	OpenAIBaseURL string
	// moonshot api 密钥
	MoonshotAPIKey string
	// ai 模型
	AIModel string
	// resume ai 最大字符数
	ResumeAIMaxRunes int
	// resume ai 最大完成令牌数
	ResumeAIMaxCompletionTokens int64
	// resume ai 温度
	ResumeAITemperature float64
}

// 验证 openai 配置
func validateOpenAIConfig() (*OpenAIConfig, error) {
	// openai api 密钥
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		return nil, errors.New(errmsg.ConfigOpenAIAPIKeyRequired)
	}
	// openai 基础 URL
	openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseURL == "" {
		return nil, errors.New(errmsg.ConfigOpenAIBaseURLRequired)
	}
	// moonshot api 密钥
	moonshotAPIKey := os.Getenv("MOONSHOT_API_KEY")
	if moonshotAPIKey == "" {
		return nil, errors.New(errmsg.ConfigMoonshotAPIKeyRequired)
	}
	// ai 模型
	aiModel := os.Getenv("AI_MODEL")
	if aiModel == "" {
		return nil, errors.New(errmsg.ConfigAIModelRequired)
	}
	// 简历侧截断：rune 上限；不填或空串则使用 defaultResumeAIMaxRunes（200K 档位，与 200K context 用法对齐）
	resumeAIMaxRunes, err := getenvPositiveInt("RESUME_AI_MAX_RUNES", defaultResumeAIMaxRunes, errmsg.ConfigResumeAIMaxRunesInvalid)
	if err != nil {
		return nil, err
	}
	// 模型输出：MaxCompletionTokens；不填或空串则 default（200K 档位，对应控制台 Maximum Output Tokens 一类）
	resumeAIMaxCompletionTokens, err := getenvPositiveInt64("RESUME_AI_MAX_COMPLETION_TOKENS", defaultResumeAIMaxCompletionTokens, errmsg.ConfigResumeAIMaxTokensInvalid)
	if err != nil {
		return nil, err
	}
	// resume ai 温度
	resumeAITemperature, err := strconv.ParseFloat(os.Getenv("RESUME_AI_TEMPERATURE"), 64)
	if err != nil || resumeAITemperature < 0 {
		return nil, errors.New(errmsg.ConfigResumeAITemperatureInvalid)
	}

	return &OpenAIConfig{
		OpenAIAPIKey:                openaiAPIKey,
		OpenAIBaseURL:               openaiBaseURL,
		MoonshotAPIKey:              moonshotAPIKey,
		AIModel:                     aiModel,
		ResumeAIMaxRunes:            resumeAIMaxRunes,
		ResumeAIMaxCompletionTokens: resumeAIMaxCompletionTokens,
		ResumeAITemperature:         resumeAITemperature,
	}, nil
}

// OpenAIConfigured 是否启用 OpenAI（OPENAI_API_KEY 非空）。
func (c *OpenAIConfig) OpenAIConfigured() bool {
	return c.OpenAIAPIKey != "" || c.OpenAIBaseURL != "" || c.MoonshotAPIKey != ""
}

func getenvPositiveInt(name string, defaultVal int, invalidMsg string) (int, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultVal, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return 0, errors.New(invalidMsg)
	}
	return i, nil
}

func getenvPositiveInt64(name string, defaultVal int64, invalidMsg string) (int64, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultVal, nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil || i <= 0 {
		return 0, errors.New(invalidMsg)
	}
	return i, nil
}
