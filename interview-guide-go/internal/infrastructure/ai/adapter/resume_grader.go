package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"unicode/utf8"

	aicore "interview-guide-go/internal/infrastructure/ai"
	"interview-guide-go/internal/infrastructure/ai/promptprofile"
	"interview-guide-go/shared/logmsg"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	constpkg "github.com/openai/openai-go/shared/constant"
	"go.uber.org/zap"
)

// 缓存 prompts 模板
type cachedPrompts struct {
	system string
	user   string
}

// 缓存 prompts 模板
var promptTextCache sync.Map

// 分析分数
type AnalysisScores struct {
	Overall         int
	Content         int
	Structure       int
	SkillMatch      int
	Expression      int
	Project         int
	Summary         string
	Strengths       []string
	SuggestionsJSON string // 对象数组原始 JSON，直接落 suggestions_json
}

// 原始简历分析
type rawResumeAnalysis struct {
	OverallScore int `json:"overallScore"`
	ScoreDetail  struct {
		ContentScore    int `json:"contentScore"`
		StructureScore  int `json:"structureScore"`
		SkillMatchScore int `json:"skillMatchScore"`
		ExpressionScore int `json:"expressionScore"`
		ProjectScore    int `json:"projectScore"`
	} `json:"scoreDetail"`
	Summary     string           `json:"summary"`
	Strengths   []string         `json:"strengths"`
	Suggestions []map[string]any `json:"suggestions"`
}

// ResumeGrader 使用 OpenAI 兼容 Chat Completions（GLM / Kimi / DashScope / OpenAI 等）。
type ResumeGrader struct {
	client              openai.Client
	model               shared.ChatModel
	maxRunes            int // 简历正文截断上限（rune）；与约 200K token 上下文模型对齐时可用 200000
	maxCompletionTokens int64
	temperature         float64 // 0 表示请求体不传该字段
	lg                  *zap.Logger
}

// NewResumeGrader 创建简历分析器。
// model 示例：glm-5.1；temperature 传 1 可兼容默认随机性为 1 的模型；0 则省略 temperature。
//
// lg 可为 nil，此时不向日志输出截断等运行信息（等价 zap.NewNop()）。
func NewResumeGrader(client openai.Client, model string, maxRunes int, maxCompletionTokens int64, temperature float64, lg *zap.Logger) *ResumeGrader {
	if maxRunes <= 0 {
		maxRunes = 200000
	}
	// 与常见「200K context / 200K max output」档位对齐；仍受模型与账户实际上限约束。
	if maxCompletionTokens < 200000 || maxCompletionTokens > 200000 {
		maxCompletionTokens = 200000
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	return &ResumeGrader{
		client:              client,
		model:               shared.ChatModel(model),
		maxRunes:            maxRunes,
		maxCompletionTokens: maxCompletionTokens,
		temperature:         temperature,
		lg:                  lg,
	}
}

// loadResumePromptPair 加载简历分析提示词对
func loadResumePromptPair(interviewerRole string) (system string, userTemplate string, err error) {
	sub := promptprofile.PromptSubdir(interviewerRole)
	if v, ok := promptTextCache.Load(sub); ok {
		c := v.(cachedPrompts)
		return c.system, c.user, nil
	}
	sysPath := fmt.Sprintf("prompts/resume/%s/resume-analysis-system.st", sub)
	userPath := fmt.Sprintf("prompts/resume/%s/resume-analysis-user.st", sub)

	// 读取系统提示词
	sysBytes, err := fs.ReadFile(aicore.PromptsRoot, sysPath)
	if err != nil {
		return "", "", fmt.Errorf("read system prompt %s: %w", sysPath, err)
	}
	userBytes, err := fs.ReadFile(aicore.PromptsRoot, userPath)
	if err != nil {
		return "", "", fmt.Errorf("read user prompt %s: %w", userPath, err)
	}
	system = string(sysBytes)
	userTemplate = string(userBytes)
	promptTextCache.Store(sub, cachedPrompts{system: system, user: userTemplate})
	return system, userTemplate, nil
}

// Grade 调用 LLM 并解析 JSON；ctx 用于取消与超时（建议调用方带 timeout）。
// interviewerRole 为 promptprofile 常量（BACKEND / FRONTEND），决定加载 prompts/resume/<subdir>/ 下模板。
func (g *ResumeGrader) Grade(ctx context.Context, resumeText string, interviewerRole string) (*AnalysisScores, error) {
	// 去除空格
	text := strings.TrimSpace(resumeText)
	if text == "" {
		return nil, fmt.Errorf("empty resume text")
	}
	// 如果文本长度超过最大长度，则截取最大长度
	if n := utf8.RuneCountInString(text); n > g.maxRunes {
		g.lg.Warn(logmsg.MsgResumeGradeTextTruncated,
			zap.Int(logmsg.FieldRuneCount, n),
			zap.Int(logmsg.FieldMaxRunes, g.maxRunes))
		text = string([]rune(text)[:g.maxRunes])
	}
	// 加载简历分析提示词对 prompts
	resumeSystemPrompt, resumeUserPromptTemplate, err := loadResumePromptPair(interviewerRole)
	if err != nil {
		return nil, err
	}

	// 替换用户提示词中的 {resumeText} 为简历文本
	userPrompt := strings.ReplaceAll(resumeUserPromptTemplate, "{resumeText}", text)

	// 创建 OpenAI 兼容 Chat Completions 请求参数
	params := openai.ChatCompletionNewParams{
		Model:               g.model,
		MaxCompletionTokens: openai.Int(g.maxCompletionTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(resumeSystemPrompt),
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
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: aicore.PtrJSONObjectFormat(),
		},
	}

	// 如果温度大于 0，则设置温度
	if g.temperature > 0 {
		params.Temperature = openai.Float(g.temperature)
	}
	// 调用 OpenAI 兼容 Chat Completions
	resp, err := g.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	// 如果没有完成选择，则返回错误
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices")
	}
	// 获取原始响应内容
	raw := strings.TrimSpace(resp.Choices[0].Message.Content)

	// 提取 JSON 对象
	raw = aicore.ExtractJSONObject(raw)

	// 解析 JSON 对象
	var parsed rawResumeAnalysis
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse model json: %w", err)
	}
	// 序列化建议
	sugBytes, err := json.Marshal(parsed.Suggestions)
	if err != nil {
		return nil, fmt.Errorf("marshal suggestions: %w", err)
	}

	// 获取 strengths
	st := parsed.Strengths
	if st == nil {
		st = []string{}
	}

	// 返回分析分数
	return &AnalysisScores{
		Overall:         parsed.OverallScore,
		Content:         parsed.ScoreDetail.ContentScore,
		Structure:       parsed.ScoreDetail.StructureScore,
		SkillMatch:      parsed.ScoreDetail.SkillMatchScore,
		Expression:      parsed.ScoreDetail.ExpressionScore,
		Project:         parsed.ScoreDetail.ProjectScore,
		Summary:         parsed.Summary,
		Strengths:       st,
		SuggestionsJSON: string(sugBytes),
	}, nil
}
