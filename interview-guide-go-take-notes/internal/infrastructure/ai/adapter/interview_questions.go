package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	aicore "interview-guide-go/internal/infrastructure/ai"
	res "interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/infrastructure/ai/promptprofile"
	ityp "interview-guide-go/shared/interview"
	"interview-guide-go/shared/logmsg"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	constpkg "github.com/openai/openai-go/shared/constant"
	"go.uber.org/zap"
)

type llmQuestionItem struct {
	// 问题
	Question string `json:"question"`
	// 问题类型
	Type string `json:"type"`
	// 问题分类
	Category string `json:"category"`
	// 追问
	FollowUps []string `json:"followUps"`
}

type llmQuestionList struct {
	Questions []llmQuestionItem `json:"questions"`
}

var interviewPromptCache sync.Map // key "<subdir>/<name>" -> string

// InterviewQuestionGenerator 对齐示例 InterviewQuestionService：简历 + 历史题 → 主问题 + 追问展开为列表。
// 生成面试题列表
type InterviewQuestionGenerator struct {
	client              openai.Client
	model               shared.ChatModel
	maxRunes            int
	maxCompletionTokens int64
	temperature         float64
	followUpCount       int
	lg                  *zap.Logger
}

// NewInterviewQuestionGenerator followUpCount 与示例 app.interview.follow-up-count 一致，建议 0～2。
func NewInterviewQuestionGenerator(client openai.Client, model string, maxRunes int, maxCompletionTokens int64, temperature float64, followUpCount int, lg *zap.Logger) *InterviewQuestionGenerator {
	if maxRunes <= 0 {
		maxRunes = 120000
	}
	if maxCompletionTokens < 2048 {
		maxCompletionTokens = 8192
	}
	if followUpCount < 0 {
		followUpCount = 0
	}
	if followUpCount > 2 {
		followUpCount = 2
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	return &InterviewQuestionGenerator{
		client:              client,
		model:               shared.ChatModel(model),
		maxRunes:            maxRunes,
		maxCompletionTokens: maxCompletionTokens,
		temperature:         temperature,
		followUpCount:       followUpCount,
		lg:                  lg,
	}
}

type questionDistribution struct {
	values map[string]int
}

// 与示例 InterviewQuestionService.calculateDistribution 同口径（比例略有取整差异）。
func calculateQuestionDistribution(total int, interviewerRole string) questionDistribution {
	if total < 1 {
		total = 1
	}
	if strings.EqualFold(strings.TrimSpace(interviewerRole), promptprofile.Backend) {
		const (
			projectRatio        = 0.20
			mysqlRatio          = 0.20
			redisRatio          = 0.20
			javaBasicRatio      = 0.10
			javaCollectionRatio = 0.10
			javaConcurrentRatio = 0.10
		)
		project := max(1, int(float64(total)*projectRatio+0.5))
		mysql := max(1, int(float64(total)*mysqlRatio+0.5))
		redis := max(1, int(float64(total)*redisRatio+0.5))
		javaBasic := max(1, int(float64(total)*javaBasicRatio+0.5))
		javaCollection := int(float64(total)*javaCollectionRatio + 0.5)
		javaConcurrent := int(float64(total)*javaConcurrentRatio + 0.5)
		spring := total - project - mysql - redis - javaBasic - javaCollection - javaConcurrent
		if spring < 0 {
			spring = 0
		}
		return questionDistribution{
			values: map[string]int{
				"projectCount":        project,
				"mysqlCount":          mysql,
				"redisCount":          redis,
				"javaBasicCount":      javaBasic,
				"javaCollectionCount": javaCollection,
				"javaConcurrentCount": javaConcurrent,
				"springCount":         spring,
			},
		}
	}
	const (
		projectRatio        = 0.20
		webBasicRatio       = 0.20
		jsTsRatio           = 0.20
		frameworkRatio      = 0.15
		browserNetworkRatio = 0.15
		engineeringRatio    = 0.10
	)
	project := max(1, int(float64(total)*projectRatio+0.5))
	webBasic := max(1, int(float64(total)*webBasicRatio+0.5))
	jsTs := max(1, int(float64(total)*jsTsRatio+0.5))
	framework := max(1, int(float64(total)*frameworkRatio+0.5))
	browserNetwork := int(float64(total)*browserNetworkRatio + 0.5)
	engineering := total - project - webBasic - jsTs - framework - browserNetwork
	if engineering < 0 {
		engineering = max(0, int(float64(total)*engineeringRatio+0.5))
	}
	return questionDistribution{
		values: map[string]int{
			"projectCount":        project,
			"webBasicCount":       webBasic,
			"jsTsCount":           jsTs,
			"frameworkCount":      framework,
			"browserNetworkCount": browserNetwork,
			"engineeringCount":    engineering,
		},
	}
}

func loadInterviewPrompt(interviewerRole, name string) (string, error) {
	subdir := promptprofile.PromptSubdir(interviewerRole)
	cacheKey := subdir + "/" + name
	if v, ok := interviewPromptCache.Load(cacheKey); ok {
		return v.(string), nil
	}
	b, err := fs.ReadFile(aicore.PromptsRoot, "prompts/interview/"+subdir+"/"+name)
	if err != nil {
		// fallback to the legacy flat path to tolerate partial upgrades
		b, err = fs.ReadFile(aicore.PromptsRoot, "prompts/interview/"+name)
		if err != nil {
			return "", err
		}
	}
	s := string(b)
	interviewPromptCache.Store(cacheKey, s)
	return s, nil
}

// GenerateForQueue 供 Redis 队列消费者使用：不在此处静默降级默认题。
// LLM 失败（含超时、取消、解析错误）或返回空列表时返回 error，由消费者将会话标记为 QUESTIONS_FAILED，
// 避免前端轮询已超时放弃而后台仍将会话写成 CREATED 的业务反差。
func (g *InterviewQuestionGenerator) GenerateForQueue(ctx context.Context, resumeText string, questionCount int, historicalQuestions []string, interviewerRole string) ([]res.InterviewQuestion, error) {
	if g == nil {
		return nil, fmt.Errorf("interview question generator is nil")
	}
	text := strings.TrimSpace(resumeText)
	if text == "" {
		return nil, fmt.Errorf("resume text is empty")
	}
	if questionCount < 1 {
		questionCount = 1
	}
	if questionCount > 30 {
		questionCount = 30
	}
	// 截断简历文本
	if n := utf8.RuneCountInString(text); n > g.maxRunes {
		g.lg.Warn("interview question resume text truncated", zap.Int("runes", n), zap.Int("max", g.maxRunes))
		text = string([]rune(text)[:g.maxRunes])
	}
	// 生成面试题列表（与主项目 Generate 一致打耗时/成败日志；此处不降级默认题，失败仅记录并返回 error）
	llmStart := time.Now()
	out, err := g.generateViaLLM(ctx, text, questionCount, historicalQuestions, interviewerRole)
	llmElapsed := time.Since(llmStart)
	if err != nil {
		g.lg.Warn(logmsg.MsgInterviewQuestionLLMFailed, zap.Error(err), zap.Duration(logmsg.FieldLLMDuration, llmElapsed))
		return nil, err
	}
	if len(out) == 0 {
		g.lg.Warn(logmsg.MsgInterviewQuestionLLMEmpty, zap.Duration(logmsg.FieldLLMDuration, llmElapsed))
		return nil, fmt.Errorf("LLM returned no questions")
	}
	g.lg.Info(logmsg.MsgInterviewQuestionLLMOK, zap.Int("questionCount", len(out)), zap.Duration(logmsg.FieldLLMDuration, llmElapsed))
	return out, nil
}

// AI 调用 LLM 生成面试题列表
func (g *InterviewQuestionGenerator) generateViaLLM(ctx context.Context, resumeText string, questionCount int, historicalQuestions []string, interviewerRole string) ([]res.InterviewQuestion, error) {
	// 解析面试官角色
	role, _ := promptprofile.Parse(interviewerRole)
	// 加载系统提示词
	sys, err := loadInterviewPrompt(role, "interview-question-system.st")
	if err != nil {
		return nil, err
	}
	// 加载用户提示词
	userTpl, err := loadInterviewPrompt(role, "interview-question-user.st")
	if err != nil {
		return nil, err
	}
	// 计算题目分布
	dist := calculateQuestionDistribution(questionCount, role)
	// 历史提问
	hist := "暂无历史提问"
	if len(historicalQuestions) > 0 {
		hist = strings.Join(historicalQuestions, "\n")
	}
	// 填充用户提示词
	user := userTpl
	// 替换变量
	repl := []struct{ old, new string }{
		{"{questionCount}", fmt.Sprint(questionCount)},
		{"{followUpCount}", fmt.Sprint(g.followUpCount)},
		{"{resumeText}", resumeText},
		{"{historicalQuestions}", hist},
	}
	// 替换题目分布
	for k, v := range dist.values {
		repl = append(repl, struct{ old, new string }{
			old: "{" + k + "}",
			new: fmt.Sprint(v),
		})
	}
	// 替换变量
	for _, p := range repl {
		user = strings.ReplaceAll(user, p.old, p.new)
	}
	// 创建 OpenAI 兼容 Chat Completions 请求参数
	params := openai.ChatCompletionNewParams{
		Model:               g.model,
		MaxCompletionTokens: openai.Int(g.maxCompletionTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(sys),
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Role: constpkg.User("user"),
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(user),
					},
				},
			},
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: aicore.PtrJSONObjectFormat(),
		},
	}
	if g.temperature > 0 {
		params.Temperature = openai.Float(g.temperature)
	}
	g.lg.Info(logmsg.MsgInterviewQuestionLLMBegin,
		zap.String(logmsg.FieldModel, string(g.model)),
		zap.String(logmsg.FieldInterviewerRole, role),
		zap.Int("questionCount", questionCount),
		zap.Int("resumeRuneCount", utf8.RuneCountInString(resumeText)),
	)
	resp, err := g.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices")
	}
	// 提取 JSON 对象
	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	// 提取 JSON 对象
	raw = aicore.ExtractJSONObject(raw)
	// 解析 JSON 对象
	var parsed llmQuestionList
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse interview questions json: %w", err)
	}

	return expandLLMQuestions(parsed, g.followUpCount, role), nil
}

// 组合 LLM 生成的题目为 InterviewQuestion 列表
func expandLLMQuestions(parsed llmQuestionList, followUpCap int, interviewerRole string) []res.InterviewQuestion {
	var out []res.InterviewQuestion
	idx := 0
	for _, q := range parsed.Questions {
		qtext := strings.TrimSpace(q.Question)
		if qtext == "" {
			continue
		}
		t := normalizeQuestionType(q.Type, interviewerRole)
		cat := strings.TrimSpace(q.Category)
		if cat == "" {
			cat = "综合"
		}
		mainIdx := idx
		out = append(out, res.InterviewQuestion{
			QuestionIndex: idx,
			Question:      qtext,
			Type:          t,
			Category:      cat,
			IsFollowUp:    false,
		})
		idx++
		fups := sanitizeFollowUps(q.FollowUps, followUpCap)
		for i, fu := range fups {
			p := mainIdx
			out = append(out, res.InterviewQuestion{
				QuestionIndex:       idx,
				Question:            fu,
				Type:                t,
				Category:            buildFollowUpCategory(cat, i+1),
				IsFollowUp:          true,
				ParentQuestionIndex: &p,
			})
			idx++
		}
	}
	return out
}

// 清理追问
func sanitizeFollowUps(in []string, cap int) []string {
	if cap <= 0 || len(in) == 0 {
		return nil
	}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= cap {
			break
		}
	}
	return out
}

// 构建追问分类
func buildFollowUpCategory(category string, order int) string {
	base := strings.TrimSpace(category)
	if base == "" {
		base = "追问"
	}
	return fmt.Sprintf("%s（追问%d）", base, order)
}

// 标准化问题类型
func normalizeQuestionType(s, interviewerRole string) ityp.QuestionType {
	u := strings.ToUpper(strings.TrimSpace(s))
	switch u {
	case "PROJECT":
		return ityp.TypeProject
	case "JAVA_BASIC":
		return ityp.TypeJavaBasic
	case "JAVA_COLLECTION":
		return ityp.TypeJavaCollection
	case "JAVA_CONCURRENT":
		return ityp.TypeJavaConcurrent
	case "MYSQL":
		return ityp.TypeMySQL
	case "REDIS":
		return ityp.TypeRedis
	case "SPRING":
		return ityp.TypeSpring
	case "SPRING_BOOT", "SPRINGBOOT":
		return ityp.TypeSpringBoot
	case "WEB_BASIC":
		return ityp.TypeWebBasic
	case "JAVASCRIPT_TYPESCRIPT":
		return ityp.TypeJavaScriptTypeScript
	case "FRAMEWORK":
		return ityp.TypeFramework
	case "BROWSER_NETWORK":
		return ityp.TypeBrowserNetwork
	case "ENGINEERING":
		return ityp.TypeEngineering
	default:
		if strings.EqualFold(strings.TrimSpace(interviewerRole), promptprofile.Backend) {
			return ityp.TypeJavaBasic
		}
		return ityp.TypeWebBasic
	}
}

func roleSpecificDefaultRows(interviewerRole string) [][3]string {
	// 后端角色专属默认题库
	if strings.EqualFold(strings.TrimSpace(interviewerRole), promptprofile.Backend) {
		return [][3]string{
			{"请介绍一下你在简历中提到的最重要的项目，你在其中承担了什么角色？", "PROJECT", "项目经历"},
			{"MySQL的索引有哪些类型？B+树索引的原理是什么？", "MYSQL", "MySQL"},
			{"Redis支持哪些数据结构？各自的使用场景是什么？", "REDIS", "Redis"},
			{"Java中HashMap的底层实现原理是什么？JDK8做了哪些优化？", "JAVA_COLLECTION", "Java集合"},
			{"synchronized和ReentrantLock有什么区别？", "JAVA_CONCURRENT", "Java并发"},
			{"Spring的IoC和AOP原理是什么？", "SPRING", "Spring"},
			{"MySQL事务的ACID特性是什么？隔离级别有哪些？", "MYSQL", "MySQL"},
			{"Redis的持久化机制有哪些？RDB和AOF的区别？", "REDIS", "Redis"},
			{"Java的垃圾回收机制是怎样的？常见的GC算法有哪些？", "JAVA_BASIC", "Java基础"},
			{"线程池的核心参数有哪些？如何合理配置？", "JAVA_CONCURRENT", "Java并发"},
		}
	}

	// 前端角色专属默认题库
	return [][3]string{
		{"请介绍一下你在简历中提到的最重要的前端或客户端项目，你在其中承担了什么角色？", "PROJECT", "项目经历"},
		{"浏览器从输入 URL 到页面可交互，中间经历了哪些关键过程？", "BROWSER_NETWORK", "浏览器与网络"},
		{"JavaScript 中原型链、作用域链和闭包分别解决了什么问题？", "JAVASCRIPT_TYPESCRIPT", "JavaScript / TypeScript"},
		{"React 或 Vue 的响应式更新机制是怎样工作的？", "FRAMEWORK", "框架"},
		{"你在项目里做过哪些性能优化？如何衡量优化是否生效？", "ENGINEERING", "工程化"},
		{"TypeScript 的泛型、类型收窄和联合类型在你的项目里怎么落地？", "JAVASCRIPT_TYPESCRIPT", "JavaScript / TypeScript"},
		{"浏览器缓存有哪些层级？强缓存和协商缓存分别适合什么场景？", "WEB_BASIC", "Web基础"},
		{"前端工程化里你如何处理构建速度、代码分包和发布回滚？", "ENGINEERING", "工程化"},
		{"如果页面出现长任务、卡顿或白屏，你会怎么定位问题？", "BROWSER_NETWORK", "浏览器与网络"},
		{"组件设计时你如何平衡复用性、可维护性和业务定制化？", "FRAMEWORK", "框架"},
	}
}

// 默认题库与示例 InterviewQuestionService.generateDefaultQuestions 主问题一致；追问文案对齐示例。
func (g *InterviewQuestionGenerator) defaultQuestions(count int, interviewerRole string) []res.InterviewQuestion {
	// 获取默认题库
	defaultRows := roleSpecificDefaultRows(interviewerRole)
	n := min(len(defaultRows), max(1, count))
	fc := g.followUpCount
	if fc < 0 {
		fc = 0
	}
	if fc > 2 {
		fc = 2
	}

	var out []res.InterviewQuestion
	idx := 0
	for i := 0; i < n; i++ {
		row := defaultRows[i]
		mainQ := row[0]
		typStr := row[1]
		cat := row[2]
		qt := normalizeQuestionType(typStr, interviewerRole)
		mainIdx := idx
		out = append(out, res.InterviewQuestion{
			QuestionIndex: idx,
			Question:      mainQ,
			Type:          qt,
			Category:      cat,
			IsFollowUp:    false,
		})
		idx++
		for j := 0; j < fc; j++ {
			p := mainIdx
			out = append(out, res.InterviewQuestion{
				QuestionIndex:       idx,
				Question:            buildDefaultFollowUp(mainQ, j+1),
				Type:                qt,
				Category:            buildFollowUpCategory(cat, j+1),
				IsFollowUp:          true,
				ParentQuestionIndex: &p,
			})
			idx++
		}
	}
	return out
}

// 构建默认追问
func buildDefaultFollowUp(mainQuestion string, order int) string {
	if order == 1 {
		return fmt.Sprintf("基于「%s」，请结合你亲自做过的一个真实场景展开说明。", mainQuestion)
	}
	return fmt.Sprintf("基于「%s」，如果线上出现异常，你会如何定位并给出修复方案？", mainQuestion)
}
