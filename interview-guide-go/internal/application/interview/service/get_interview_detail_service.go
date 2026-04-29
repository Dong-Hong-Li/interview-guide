package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// GetInterviewDetailService GET /api/interview/sessions/{id}/details（与主项目 GetDetail 一致：仅校验会话存在，无交卷/报告门禁）。
type GetInterviewDetailService struct {
	sessions repository.InterviewSessionWriter
}

func NewGetInterviewDetailService(sessions repository.InterviewSessionWriter) *GetInterviewDetailService {
	return &GetInterviewDetailService{sessions: sessions}
}

// GetDetail sessionID 须已由 controller Trim。
func (s *GetInterviewDetailService) GetDetail(ctx context.Context, sessionID string) (*results.InterviewDetail, error) {
	if s.sessions == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview detail not configured")
	}
	sess, answers, err := s.sessions.LoadForReport(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}
	out := buildInterviewDetail(sess, answers)
	return &out, nil
}

func buildInterviewDetail(sess *results.SessionReportDB, answers []results.InterviewAnswerDB) results.InterviewDetail {
	if sess == nil {
		return results.InterviewDetail{}
	}
	var strengths []string
	if raw := strings.TrimSpace(sess.StrengthsJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &strengths)
	}
	if strengths == nil {
		strengths = []string{}
	}
	var improvements []string
	if raw := strings.TrimSpace(sess.ImprovementsJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &improvements)
	}
	if improvements == nil {
		improvements = []string{}
	}
	var refAnswers []any
	if raw := strings.TrimSpace(sess.ReferenceAnswersJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &refAnswers)
	}
	if refAnswers == nil {
		refAnswers = []any{}
	}
	var questionsAny []any
	if raw := strings.TrimSpace(sess.QuestionsJSON); raw != "" {
		_ = json.Unmarshal([]byte(raw), &questionsAny)
	}
	if questionsAny == nil {
		questionsAny = []any{}
	}
	var qs []results.InterviewQuestion
	_ = json.Unmarshal([]byte(strings.TrimSpace(sess.QuestionsJSON)), &qs)

	st := strings.TrimSpace(sess.Status)
	if st == "" {
		st = "CREATED"
	}
	return results.InterviewDetail{
		ID:               sess.InternalID,
		SessionID:        sess.SessionID,
		TotalQuestions:   sess.TotalQuestions,
		Status:           st,
		EvaluateStatus:   strings.TrimSpace(sess.EvaluateStatus),
		EvaluateError:    strings.TrimSpace(sess.EvaluateError),
		OverallScore:     sess.OverallScore,
		OverallFeedback:  strings.TrimSpace(sess.OverallFeedback),
		CreatedAt:        sess.CreatedAt,
		CompletedAt:      sess.CompletedAt,
		Questions:        questionsAny,
		Strengths:        strengths,
		Improvements:     improvements,
		ReferenceAnswers: refAnswers,
		Answers:          buildDetailAnswerList(qs, answers),
	}
}

func buildDetailAnswerList(qs []results.InterviewQuestion, answers []results.InterviewAnswerDB) []results.InterviewAnswerItem {
	ansByIdx := make(map[int]results.InterviewAnswerDB, len(answers))
	for _, a := range answers {
		ansByIdx[a.QuestionIndex] = a
	}
	if len(qs) == 0 {
		return interviewAnswerItemsFromDBOnly(answers)
	}
	out := make([]results.InterviewAnswerItem, 0, len(qs))
	for _, q := range qs {
		a, has := ansByIdx[q.QuestionIndex]
		score := 0
		feedback := feedbackFromQuestion(q)
		userAnswer := ""
		var answeredAt *time.Time
		ref := ""
		var keyPoints []string
		if has {
			userAnswer = strings.TrimSpace(a.UserAnswer)
			if a.Score != nil {
				score = *a.Score
			} else if q.Score != nil {
				score = *q.Score
			}
			if strings.TrimSpace(a.Feedback) != "" {
				feedback = strings.TrimSpace(a.Feedback)
			}
			if !a.AnsweredAt.IsZero() {
				t := a.AnsweredAt
				answeredAt = &t
			}
			ref = strings.TrimSpace(a.ReferenceAnswer)
			keyPoints = parseKeyPointsJSON(a.KeyPointsJSON)
		} else if q.Score != nil {
			score = *q.Score
		}
		out = append(out, results.InterviewAnswerItem{
			QuestionIndex:   q.QuestionIndex,
			Question:        strings.TrimSpace(q.Question),
			Category:        strings.TrimSpace(q.Category),
			UserAnswer:      userAnswer,
			Score:           score,
			Feedback:        feedback,
			ReferenceAnswer: ref,
			KeyPoints:       keyPoints,
			AnsweredAt:      answeredAt,
		})
	}
	return out
}

func feedbackFromQuestion(q results.InterviewQuestion) string {
	if q.Feedback != nil {
		return strings.TrimSpace(*q.Feedback)
	}
	return ""
}

func interviewAnswerItemsFromDBOnly(answers []results.InterviewAnswerDB) []results.InterviewAnswerItem {
	if len(answers) == 0 {
		return []results.InterviewAnswerItem{}
	}
	sorted := append([]results.InterviewAnswerDB(nil), answers...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].QuestionIndex < sorted[j].QuestionIndex
	})
	out := make([]results.InterviewAnswerItem, 0, len(sorted))
	for _, a := range sorted {
		sc := 0
		if a.Score != nil {
			sc = *a.Score
		}
		var atPtr *time.Time
		if !a.AnsweredAt.IsZero() {
			t := a.AnsweredAt
			atPtr = &t
		}
		out = append(out, results.InterviewAnswerItem{
			QuestionIndex:   a.QuestionIndex,
			Question:        strings.TrimSpace(a.Question),
			Category:        strings.TrimSpace(a.Category),
			UserAnswer:      strings.TrimSpace(a.UserAnswer),
			Score:           sc,
			Feedback:        strings.TrimSpace(a.Feedback),
			ReferenceAnswer: strings.TrimSpace(a.ReferenceAnswer),
			KeyPoints:       parseKeyPointsJSON(a.KeyPointsJSON),
			AnsweredAt:      atPtr,
		})
	}
	return out
}

func parseKeyPointsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var pts []string
	if err := json.Unmarshal([]byte(raw), &pts); err != nil {
		return nil
	}
	return pts
}
