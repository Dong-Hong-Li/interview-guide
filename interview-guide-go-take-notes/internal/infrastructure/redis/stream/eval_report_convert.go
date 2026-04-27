package redisstream

import (
	ivmodel "interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/interfaces/api/dto"
)

func evaluationReportFromDTO(r *dto.InterviewReport) *ivmodel.EvaluationReport {
	if r == nil {
		return nil
	}
	details := make([]ivmodel.EvaluationQuestionDetail, 0, len(r.QuestionDetails))
	for _, q := range r.QuestionDetails {
		details = append(details, ivmodel.EvaluationQuestionDetail{
			QuestionIndex: q.QuestionIndex,
			Question:      q.Question,
			Category:      q.Category,
			UserAnswer:    q.UserAnswer,
			Score:         q.Score,
			Feedback:      q.Feedback,
		})
	}
	refs := make([]ivmodel.EvaluationReferenceAnswer, 0, len(r.ReferenceAnswers))
	for _, x := range r.ReferenceAnswers {
		kp := x.KeyPoints
		if kp == nil {
			kp = []string{}
		}
		refs = append(refs, ivmodel.EvaluationReferenceAnswer{
			QuestionIndex:   x.QuestionIndex,
			Question:        x.Question,
			ReferenceAnswer: x.ReferenceAnswer,
			KeyPoints:       kp,
		})
	}
	st := r.Strengths
	if st == nil {
		st = []string{}
	}
	im := r.Improvements
	if im == nil {
		im = []string{}
	}
	return &ivmodel.EvaluationReport{
		OverallScore:     r.OverallScore,
		OverallFeedback:  r.OverallFeedback,
		Strengths:        st,
		Improvements:     im,
		QuestionDetails:  details,
		ReferenceAnswers: refs,
	}
}
