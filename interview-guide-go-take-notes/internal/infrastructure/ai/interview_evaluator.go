package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"interview-guide-go/shared/logmsg"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	constpkg "github.com/openai/openai-go/shared/constant"
	"go.uber.org/zap"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/infrastructure/ai/promptprofile"
)

// 与 Java AnswerEvaluationService.EvaluationReportDTO + BeanOutputConverter 对齐的批次 JSON 契约（非完整 InterviewReport）。
const interviewBatchEvalJSONContract = `

# 机器可读输出契约（覆盖上文「Output Format」中 sessionId 等字段）
请仅输出一个 JSON 对象，字段必须为：
- overallScore：整数（对本批次的初步综合分，可为近似值）
- overallFeedback：字符串
- strengths：字符串数组
- improvements：字符串数组
- questionEvaluations：对象数组，长度必须等于本批次题目数量，且顺序与输入问答记录一致；每项含：
  - questionIndex：整数（题目从 0 起的序号）
  - score：整数 0-100
  - feedback：字符串
  - referenceAnswer：字符串
  - keyPoints：字符串数组
不要输出 Markdown 代码块。`

const interviewSummaryJSONContract = `

# 机器可读输出契约
请仅输出一个 JSON 对象，字段必须为：overallFeedback（字符串）、strengths（字符串数组）、improvements（字符串数组）。不要输出 Markdown。`

type cachedInterviewEvalPrompts struct {
	evalSystem    string
	evalUser      string
	summarySystem string
	summaryUser   string
}

var interviewEvalPromptCache sync.Map

// interviewQuestionUserAnswerStr 将题目中的用户作答（JSON 反序列化后为 *string）转为用于 LLM 与报告落库的纯文本。
func interviewQuestionUserAnswerStr(q results.InterviewQuestion) string {
	if q.UserAnswer == nil {
		return ""
	}
	return strings.TrimSpace(*q.UserAnswer)
}

// InterviewEvaluator 对齐 Java AnswerEvaluationService：分批评估 + 二次汇总 + convertToReport。
type InterviewEvaluator struct {
	client              openai.Client
	model               shared.ChatModel
	maxRunes            int
	maxCompletionTokens int64
	temperature         float64
	batchSize           int
	lg                  *zap.Logger
}

// NewInterviewEvaluator 加载 prompts/interview-eval 下模板；batchSize<1 时按 8 处理。
func NewInterviewEvaluator(client openai.Client, model string, maxRunes int, maxCompletionTokens int64, temperature float64, batchSize int, lg *zap.Logger) (*InterviewEvaluator, error) {
	if maxRunes <= 0 {
		maxRunes = 120000
	}
	if maxCompletionTokens < 2048 {
		maxCompletionTokens = 32768
	}
	if maxCompletionTokens > 128000 {
		maxCompletionTokens = 128000
	}
	if batchSize < 1 {
		batchSize = 8
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	return &InterviewEvaluator{
		client:              client,
		model:               shared.ChatModel(model),
		maxRunes:            maxRunes,
		maxCompletionTokens: maxCompletionTokens,
		temperature:         temperature,
		batchSize:           batchSize,
		lg:                  lg,
	}, nil
}

func loadInterviewEvalPrompts(interviewerRole string) (cachedInterviewEvalPrompts, error) {
	sub := promptprofile.PromptSubdir(interviewerRole)
	if v, ok := interviewEvalPromptCache.Load(sub); ok {
		return v.(cachedInterviewEvalPrompts), nil
	}
	read := func(name string) (string, error) {
		path := fmt.Sprintf("prompts/interview-eval/%s/%s", sub, name)
		b, err := fs.ReadFile(promptsRoot, path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		return string(b), nil
	}
	evalSystem, err := read("interview-evaluation-system.st")
	if err != nil {
		return cachedInterviewEvalPrompts{}, err
	}
	evalUser, err := read("interview-evaluation-user.st")
	if err != nil {
		return cachedInterviewEvalPrompts{}, err
	}
	summarySystem, err := read("interview-evaluation-summary-system.st")
	if err != nil {
		return cachedInterviewEvalPrompts{}, err
	}
	summaryUser, err := read("interview-evaluation-summary-user.st")
	if err != nil {
		return cachedInterviewEvalPrompts{}, err
	}
	out := cachedInterviewEvalPrompts{
		evalSystem:    evalSystem,
		evalUser:      evalUser,
		summarySystem: summarySystem,
		summaryUser:   summaryUser,
	}
	interviewEvalPromptCache.Store(sub, out)
	return out, nil
}

type interviewBatchEvalReport struct {
	OverallScore        int                          `json:"overallScore"`
	OverallFeedback     string                       `json:"overallFeedback"`
	Strengths           []string                     `json:"strengths"`
	Improvements        []string                     `json:"improvements"`
	QuestionEvaluations []interviewBatchQuestionEval `json:"questionEvaluations"`
}

type interviewBatchQuestionEval struct {
	QuestionIndex   int      `json:"questionIndex"`
	Score           int      `json:"score"`
	Feedback        string   `json:"feedback"`
	ReferenceAnswer string   `json:"referenceAnswer"`
	KeyPoints       []string `json:"keyPoints"`
}

type interviewBatchEvalResult struct {
	start, end int
	report     interviewBatchEvalReport
}

type interviewFinalSummaryDTO struct {
	OverallFeedback string   `json:"overallFeedback"`
	Strengths       []string `json:"strengths"`
	Improvements    []string `json:"improvements"`
}

// EvaluateInterview 对整场面试打分并生成 results.InterviewReport（sessionPublicID 写入报告）。
func (e *InterviewEvaluator) EvaluateInterview(ctx context.Context, sessionPublicID, resumeText, interviewerRole string, questions []results.InterviewQuestion) (results.InterviewReport, error) {
	var empty results.InterviewReport
	if e == nil {
		return empty, fmt.Errorf("nil evaluator")
	}
	sid := strings.TrimSpace(sessionPublicID)
	if sid == "" {
		return empty, fmt.Errorf("empty session id")
	}
	if len(questions) == 0 {
		return empty, fmt.Errorf("no questions")
	}
	resumeSummary := strings.TrimSpace(resumeText)
	if len(resumeSummary) > 500 {
		resumeSummary = resumeSummary[:500] + "..."
	}
	prompts, err := loadInterviewEvalPrompts(interviewerRole)
	if err != nil {
		return empty, err
	}

	batches, err := e.evaluateInBatches(ctx, sid, resumeSummary, questions, prompts)
	if err != nil {
		return empty, err
	}
	merged := mergeInterviewQuestionEvaluations(batches)
	fallbackFB := mergeInterviewOverallFeedback(batches)
	fallbackSt := mergeInterviewListItems(batches, true)
	fallbackIm := mergeInterviewListItems(batches, false)

	final, err := e.summarizeBatchResults(ctx, resumeSummary, questions, merged, fallbackFB, fallbackSt, fallbackIm, prompts)
	if err != nil {
		e.lg.Warn(logmsg.MsgInterviewEvaluateSummaryFallback, zap.String(logmsg.FieldSessionID, sid), zap.Error(err))
		final = interviewFinalSummaryDTO{
			OverallFeedback: fallbackFB,
			Strengths:       fallbackSt,
			Improvements:    fallbackIm,
		}
	}
	return convertInterviewEvalToReport(sid, merged, questions, final.OverallFeedback, final.Strengths, final.Improvements), nil
}

func (e *InterviewEvaluator) evaluateInBatches(ctx context.Context, sessionPublicID, resumeSummary string, questions []results.InterviewQuestion, prompts cachedInterviewEvalPrompts) ([]interviewBatchEvalResult, error) {
	var out []interviewBatchEvalResult
	for start := 0; start < len(questions); start += e.batchSize {
		end := start + e.batchSize
		if end > len(questions) {
			end = len(questions)
		}
		batch := questions[start:end]
		rep, err := e.evaluateBatch(ctx, sessionPublicID, resumeSummary, batch, start, end, prompts)
		if err != nil {
			return nil, err
		}
		out = append(out, interviewBatchEvalResult{start: start, end: end, report: rep})
	}
	return out, nil
}

func buildInterviewQARecords(qs []results.InterviewQuestion) string {
	var b strings.Builder
	for _, q := range qs {
		ua := interviewQuestionUserAnswerStr(q)
		if ua == "" {
			ua = "(未回答)"
		}
		cat := strings.TrimSpace(q.Category)
		if cat == "" {
			cat = "综合"
		}
		fmt.Fprintf(&b, "问题%d [%s]: %s\n回答: %s\n\n", q.QuestionIndex+1, cat, q.Question, ua)
	}
	return b.String()
}

func (e *InterviewEvaluator) evaluateBatch(ctx context.Context, sessionPublicID, resumeSummary string, batch []results.InterviewQuestion, start, end int, prompts cachedInterviewEvalPrompts) (interviewBatchEvalReport, error) {
	var empty interviewBatchEvalReport
	qa := buildInterviewQARecords(batch)
	user := strings.ReplaceAll(prompts.evalUser, "{resumeText}", resumeSummary)
	user = strings.ReplaceAll(user, "{qaRecords}", qa)
	sys := prompts.evalSystem + interviewBatchEvalJSONContract
	if n := utf8.RuneCountInString(user); n > e.maxRunes {
		e.lg.Warn(logmsg.MsgResumeGradeTextTruncated,
			zap.Int(logmsg.FieldRuneCount, n),
			zap.Int(logmsg.FieldMaxRunes, e.maxRunes),
			zap.String(logmsg.FieldSessionID, sessionPublicID),
		)
		user = string([]rune(user)[:e.maxRunes])
	}
	raw, err := e.chatJSON(ctx, sys, user)
	if err != nil {
		return empty, fmt.Errorf("batch [%d,%d): %w", start, end, err)
	}
	raw = extractJSONObject(raw)
	var rep interviewBatchEvalReport
	if err := json.Unmarshal([]byte(raw), &rep); err != nil {
		return empty, fmt.Errorf("batch [%d,%d) parse: %w", start, end, err)
	}
	return rep, nil
}

func (e *InterviewEvaluator) chatJSON(ctx context.Context, system, user string) (string, error) {
	params := openai.ChatCompletionNewParams{
		Model:               e.model,
		MaxCompletionTokens: openai.Int(e.maxCompletionTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Role: constpkg.System("system"),
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(system),
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
			OfJSONObject: ptrJSONObjectFormat(),
		},
	}
	if e.temperature > 0 {
		params.Temperature = openai.Float(e.temperature)
	}
	resp, err := e.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func mergeInterviewQuestionEvaluations(batchResults []interviewBatchEvalResult) []interviewBatchQuestionEval {
	var merged []interviewBatchQuestionEval
	for _, br := range batchResults {
		expected := br.end - br.start
		cur := br.report.QuestionEvaluations
		for i := 0; i < expected; i++ {
			globalIdx := br.start + i
			if i < len(cur) {
				ev := cur[i]
				ev.QuestionIndex = globalIdx
				merged = append(merged, ev)
				continue
			}
			merged = append(merged, interviewBatchQuestionEval{
				QuestionIndex:   globalIdx,
				Score:           0,
				Feedback:        "该题未成功生成评估结果，系统按 0 分处理。",
				ReferenceAnswer: "",
				KeyPoints:       nil,
			})
		}
	}
	return merged
}

func mergeInterviewOverallFeedback(batchResults []interviewBatchEvalResult) string {
	var parts []string
	for _, br := range batchResults {
		fb := strings.TrimSpace(br.report.OverallFeedback)
		if fb != "" {
			parts = append(parts, fb)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n\n")
	}
	return "本次面试已完成分批评估，但未生成有效综合评语。"
}

func mergeInterviewListItems(batchResults []interviewBatchEvalResult, strengthsMode bool) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, br := range batchResults {
		var items []string
		if strengthsMode {
			items = br.report.Strengths
		} else {
			items = br.report.Improvements
		}
		for _, it := range items {
			t := strings.TrimSpace(it)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
			if len(out) >= 8 {
				return out
			}
		}
	}
	return out
}

func (e *InterviewEvaluator) summarizeBatchResults(
	ctx context.Context,
	resumeSummary string,
	questions []results.InterviewQuestion,
	evaluations []interviewBatchQuestionEval,
	fallbackFB string,
	fallbackSt, fallbackIm []string,
	prompts cachedInterviewEvalPrompts,
) (interviewFinalSummaryDTO, error) {
	catSum := buildInterviewCategorySummaryText(questions, evaluations)
	qHigh := buildInterviewQuestionHighlightsText(questions, evaluations)
	user := strings.ReplaceAll(prompts.summaryUser, "{resumeText}", resumeSummary)
	user = strings.ReplaceAll(user, "{categorySummary}", catSum)
	user = strings.ReplaceAll(user, "{questionHighlights}", qHigh)
	user = strings.ReplaceAll(user, "{fallbackOverallFeedback}", fallbackFB)
	user = strings.ReplaceAll(user, "{fallbackStrengths}", strings.Join(fallbackSt, "\n"))
	user = strings.ReplaceAll(user, "{fallbackImprovements}", strings.Join(fallbackIm, "\n"))
	sys := prompts.summarySystem + interviewSummaryJSONContract
	raw, err := e.chatJSON(ctx, sys, user)
	if err != nil {
		return interviewFinalSummaryDTO{}, err
	}
	raw = extractJSONObject(raw)
	var parsed interviewFinalSummaryDTO
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return interviewFinalSummaryDTO{}, err
	}
	fb := strings.TrimSpace(parsed.OverallFeedback)
	if fb == "" {
		fb = fallbackFB
	}
	st := sanitizeInterviewSummaryItems(parsed.Strengths, fallbackSt)
	im := sanitizeInterviewSummaryItems(parsed.Improvements, fallbackIm)
	return interviewFinalSummaryDTO{OverallFeedback: fb, Strengths: st, Improvements: im}, nil
}

func sanitizeInterviewSummaryItems(primary, fallback []string) []string {
	src := primary
	if len(src) == 0 {
		src = fallback
	}
	var out []string
	seen := make(map[string]struct{})
	for _, it := range src {
		t := strings.TrimSpace(it)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func buildInterviewCategorySummaryText(questions []results.InterviewQuestion, evaluations []interviewBatchQuestionEval) string {
	type bucket struct {
		scores []int
	}
	m := make(map[string]*bucket)
	for i, q := range questions {
		cat := strings.TrimSpace(q.Category)
		if cat == "" {
			cat = "综合"
		}
		b := m[cat]
		if b == nil {
			b = &bucket{}
			m[cat] = b
		}
		score := 0
		if i < len(evaluations) && interviewQuestionUserAnswerStr(q) != "" {
			score = evaluations[i].Score
		}
		b.scores = append(b.scores, score)
	}
	cats := make([]string, 0, len(m))
	for c := range m {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	var lines []string
	for _, c := range cats {
		scores := m[c].scores
		n := len(scores)
		sum := 0
		for _, s := range scores {
			sum += s
		}
		avg := 0
		if n > 0 {
			avg = sum / n
		}
		lines = append(lines, fmt.Sprintf("- %s: 平均分 %d, 题数 %d", c, avg, n))
	}
	return strings.Join(lines, "\n")
}

func buildInterviewQuestionHighlightsText(questions []results.InterviewQuestion, evaluations []interviewBatchQuestionEval) string {
	var lines []string
	for i, q := range questions {
		if len(lines) >= 20 {
			break
		}
		ev := interviewBatchQuestionEval{}
		if i < len(evaluations) {
			ev = evaluations[i]
		}
		qt := q.Question
		if len(qt) > 50 {
			qt = qt[:50] + "..."
		}
		fb := ev.Feedback
		if len(fb) > 80 {
			fb = fb[:80] + "..."
		}
		lines = append(lines, fmt.Sprintf("- Q%d | %s | 分数:%d | 反馈:%s",
			q.QuestionIndex+1, qt, ev.Score, fb))
	}
	return strings.Join(lines, "\n")
}

func convertInterviewEvalToReport(
	sessionPublicID string,
	evaluations []interviewBatchQuestionEval,
	questions []results.InterviewQuestion,
	overallFeedback string,
	strengths, improvements []string,
) results.InterviewReport {
	var qDetails []results.QuestionEvaluation
	var refAnswers []results.ReferenceAnswer
	catMap := make(map[string][]int)

	answered := 0
	for _, q := range questions {
		if interviewQuestionUserAnswerStr(q) != "" {
			answered++
		}
	}

	for i := 0; i < len(questions); i++ {
		q := questions[i]
		var ev *interviewBatchQuestionEval
		if i < len(evaluations) {
			ev = &evaluations[i]
		}
		feedback := "该题未成功生成评估反馈。"
		refText := ""
		var keyPts []string
		if ev != nil && strings.TrimSpace(ev.Feedback) != "" {
			feedback = ev.Feedback
		}
		if ev != nil {
			refText = ev.ReferenceAnswer
			keyPts = ev.KeyPoints
			if keyPts == nil {
				keyPts = []string{}
			}
		}
		hasAnswer := interviewQuestionUserAnswerStr(q) != ""
		score := 0
		if hasAnswer && ev != nil {
			score = ev.Score
		}
		cat := strings.TrimSpace(q.Category)
		if cat == "" {
			cat = "综合"
		}
		qDetails = append(qDetails, results.QuestionEvaluation{
			QuestionIndex: q.QuestionIndex,
			Question:      q.Question,
			Category:      cat,
			UserAnswer:    interviewQuestionUserAnswerStr(q),
			Score:         score,
			Feedback:      feedback,
		})
		refAnswers = append(refAnswers, results.ReferenceAnswer{
			QuestionIndex:   q.QuestionIndex,
			Question:        q.Question,
			ReferenceAnswer: refText,
			KeyPoints:       keyPts,
		})
		catMap[cat] = append(catMap[cat], score)
	}

	cats := make([]string, 0, len(catMap))
	for c := range catMap {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	var catScores []results.InterviewCategoryScore
	for _, c := range cats {
		arr := catMap[c]
		sum := 0
		for _, s := range arr {
			sum += s
		}
		avg := 0
		if len(arr) > 0 {
			avg = sum / len(arr)
		}
		catScores = append(catScores, results.InterviewCategoryScore{
			Category:      c,
			Score:         avg,
			QuestionCount: len(arr),
		})
	}

	overall := 0
	if answered > 0 && len(qDetails) > 0 {
		sum := 0
		for _, d := range qDetails {
			sum += d.Score
		}
		// 与 Java questionDetails.stream().mapToInt(...).average() 后 (int) 强转一致（向零截断）
		overall = sum / len(qDetails)
	}

	if strengths == nil {
		strengths = []string{}
	}
	if improvements == nil {
		improvements = []string{}
	}

	return results.InterviewReport{
		SessionID:        sessionPublicID,
		TotalQuestions:   len(questions),
		OverallScore:     overall,
		CategoryScores:   catScores,
		QuestionDetails:  qDetails,
		OverallFeedback:  overallFeedback,
		Strengths:        strengths,
		Improvements:     improvements,
		ReferenceAnswers: refAnswers,
	}
}
