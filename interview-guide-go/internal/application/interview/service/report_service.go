package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// ReportService GET /sessions/{id}/report：状态门禁 + 合成报告 + 评估未完成时给出提示与均分兜底。
type ReportService struct {
	sessions repository.InterviewSessionWriter
}

func NewReportService(sessions repository.InterviewSessionWriter) *ReportService {
	return &ReportService{sessions: sessions}
}

// GetReport 返回与前端 `InterviewReport` 同构的 body。sessionID 须已由 controller 校验非空并 Trim。
func (s *ReportService) GetReport(ctx context.Context, sessionID string) (*results.InterviewReport, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview report not configured")
	}

	// 加载会话记录
	dbSess, answers, err := s.sessions.LoadForReport(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if dbSess == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}
	if err := reportExportGate(dbSess); err != nil {
		return nil, err
	}
	// 生成综合报告提示
	synthetic := syntheticReportNotice(dbSess)

	// 解析题目
	var qs []results.InterviewQuestion
	if raw := strings.TrimSpace(dbSess.QuestionsJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &qs); err != nil {
			return nil, response.Err(http.StatusInternalServerError, "会话题目数据异常")
		}
	}
	// 构建题目评估
	details := buildQuestionEvaluations(qs, answers)
	if len(details) == 0 && len(answers) > 0 {
		details = buildQuestionEvaluationsFromAnswersOnly(answers)
	}

	// 解析优势
	var strengths []string
	if raw := strings.TrimSpace(dbSess.StrengthsJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &strengths)
	}
	// 解析改进建议
	if strengths == nil {
		strengths = []string{}
	}
	var improvements []string
	if raw := strings.TrimSpace(dbSess.ImprovementsJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &improvements)
	}
	if improvements == nil {
		improvements = []string{}
	}
	// 解析参考答案
	var refAnswers []results.ReferenceAnswer
	if raw := strings.TrimSpace(dbSess.ReferenceAnswersJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &refAnswers)
	}
	if refAnswers == nil {
		refAnswers = []results.ReferenceAnswer{}
	}

	// 计算总分
	overall := 0
	if dbSess.OverallScore != nil {
		overall = *dbSess.OverallScore
	}
	// 生成综合报告提示
	feedback := strings.TrimSpace(dbSess.OverallFeedback)
	if synthetic != "" {
		feedback = synthetic
		if overall == 0 && len(details) > 0 {
			sum := 0
			for _, d := range details {
				sum += d.Score
			}
			overall = (sum + len(details)/2) / len(details)
		}
	}

	tq := len(qs)
	if dbSess.TotalQuestions != nil && *dbSess.TotalQuestions > 0 {
		tq = *dbSess.TotalQuestions
	}

	return &results.InterviewReport{
		SessionID:        dbSess.SessionID,
		TotalQuestions:   tq,
		OverallScore:     overall,
		CategoryScores:   aggregateCategoryScores(details),
		QuestionDetails:  details,
		OverallFeedback:  feedback,
		Strengths:        strengths,
		Improvements:     improvements,
		ReferenceAnswers: refAnswers,
	}, nil
}

func reportExportGate(sess *results.SessionReportDB) error {
	st := strings.ToUpper(strings.TrimSpace(sess.Status))
	if st != "COMPLETED" && st != "EVALUATED" {
		return response.Err(http.StatusBadRequest, errmsg.GetInterviewReportNotCompleted)
	}
	if st == "COMPLETED" {
		ev := strings.ToUpper(strings.TrimSpace(sess.EvaluateStatus))
		if ev == "FAILED" {
			if msg := strings.TrimSpace(sess.EvaluateError); msg != "" {
				return response.Err(http.StatusBadRequest, msg)
			}
			return response.Err(http.StatusBadRequest, errmsg.GetInterviewReportEvalFailed)
		}
	}
	return nil
}

func syntheticReportNotice(sess *results.SessionReportDB) string {
	st := strings.ToUpper(strings.TrimSpace(sess.Status))
	ev := strings.ToUpper(strings.TrimSpace(sess.EvaluateStatus))
	if st == "COMPLETED" && ev != "COMPLETED" {
		return errmsg.GetInterviewReportSyntheticNotice
	}
	return ""
}

// buildQuestionEvaluations 构建题目评估
func buildQuestionEvaluations(qs []results.InterviewQuestion, answers []results.InterviewAnswerDB) []results.QuestionEvaluation {
	ansByIdx := make(map[int]results.InterviewAnswerDB, len(answers))
	for _, a := range answers {
		ansByIdx[a.QuestionIndex] = a
	}
	out := make([]results.QuestionEvaluation, 0, len(qs))
	for _, q := range qs {
		a, has := ansByIdx[q.QuestionIndex]
		score := 0
		fb := ""
		if q.Feedback != nil {
			fb = strings.TrimSpace(*q.Feedback)
		}
		userAnswer := ""
		if has {
			userAnswer = strings.TrimSpace(a.UserAnswer)
			if a.Score != nil {
				score = *a.Score
			} else if q.Score != nil {
				score = *q.Score
			}
			if strings.TrimSpace(a.Feedback) != "" {
				fb = strings.TrimSpace(a.Feedback)
			}
		} else if q.Score != nil {
			score = *q.Score
		}
		out = append(out, results.QuestionEvaluation{
			QuestionIndex: q.QuestionIndex,
			Question:      strings.TrimSpace(q.Question),
			Category:      strings.TrimSpace(q.Category),
			UserAnswer:    userAnswer,
			Score:         score,
			Feedback:      fb,
		})
	}
	return out
}

func buildQuestionEvaluationsFromAnswersOnly(answers []results.InterviewAnswerDB) []results.QuestionEvaluation {
	sorted := append([]results.InterviewAnswerDB(nil), answers...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].QuestionIndex < sorted[j].QuestionIndex
	})
	out := make([]results.QuestionEvaluation, 0, len(sorted))
	for _, a := range sorted {
		sc := 0
		if a.Score != nil {
			sc = *a.Score
		}
		out = append(out, results.QuestionEvaluation{
			QuestionIndex: a.QuestionIndex,
			Question:      strings.TrimSpace(a.Question),
			Category:      strings.TrimSpace(a.Category),
			UserAnswer:    strings.TrimSpace(a.UserAnswer),
			Score:         sc,
			Feedback:      strings.TrimSpace(a.Feedback),
		})
	}
	return out
}

func aggregateCategoryScores(details []results.QuestionEvaluation) []results.InterviewCategoryScore {
	type agg struct {
		sum int
		n   int
	}
	m := make(map[string]*agg)
	for _, d := range details {
		cat := strings.TrimSpace(d.Category)
		if cat == "" {
			cat = "综合"
		}
		a := m[cat]
		if a == nil {
			a = &agg{}
			m[cat] = a
		}
		a.sum += d.Score
		a.n++
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]results.InterviewCategoryScore, 0, len(keys))
	for _, c := range keys {
		a := m[c]
		avg := 0
		if a.n > 0 {
			avg = (a.sum + a.n/2) / a.n
		}
		out = append(out, results.InterviewCategoryScore{
			Category:      c,
			Score:         avg,
			QuestionCount: a.n,
		})
	}
	return out
}
