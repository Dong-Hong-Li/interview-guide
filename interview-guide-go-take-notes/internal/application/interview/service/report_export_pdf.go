package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	pdfexport "interview-guide-go/internal/infrastructure/pdf"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// ExportInterviewPDF 与主项目 GetReport/export 一致：需交卷完成且评估未失败；返回 PDF 字节与 Content-Disposition。
func (s *ReportService) ExportInterviewPDF(ctx context.Context, sessionID string) (pdfBytes []byte, contentDisposition string, err error) {
	if s.sessions == nil {
		return nil, "", response.Err(http.StatusServiceUnavailable, "interview export not configured")
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return nil, "", response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID)
	}
	sess, answers, err := s.sessions.LoadForReport(ctx, sid)
	if err != nil {
		return nil, "", err
	}
	if sess == nil {
		return nil, "", response.Err(http.StatusNotFound, errmsg.SubmitAnswerSessionNotFound)
	}
	if err := reportExportGate(sess); err != nil {
		return nil, "", err
	}

	pdfSess := &pdfexport.InterviewReportSession{
		SessionID:        sess.SessionID,
		TotalQuestions:   sess.TotalQuestions,
		Status:           sess.Status,
		OverallScore:     sess.OverallScore,
		OverallFeedback:  sess.OverallFeedback,
		StrengthsJSON:    sess.StrengthsJSON,
		ImprovementsJSON: sess.ImprovementsJSON,
		CreatedAt:        sess.CreatedAt,
		CompletedAt:      sess.CompletedAt,
	}
	pdfAns := make([]pdfexport.InterviewReportAnswer, 0, len(answers))
	for _, a := range answers {
		pdfAns = append(pdfAns, pdfexport.InterviewReportAnswer{
			QuestionIndex:   a.QuestionIndex,
			Question:        a.Question,
			Category:        a.Category,
			UserAnswer:      a.UserAnswer,
			Score:           a.Score,
			Feedback:        a.Feedback,
			ReferenceAnswer: a.ReferenceAnswer,
			KeyPointsJSON:   a.KeyPointsJSON,
		})
	}

	out, err := pdfexport.RenderInterviewReportPDF(pdfSess, pdfAns)
	if err != nil {
		if errors.Is(err, pdfexport.ErrNoFont) {
			return nil, "", response.Err(http.StatusServiceUnavailable, errmsg.PDFExportFontHint)
		}
		return nil, "", response.Err(http.StatusInternalServerError, errmsg.InterviewExportPDFFailed)
	}
	fn := pdfexport.InterviewExportFilename(sid)
	return out, pdfexport.ContentDispositionRFC5987(fn), nil
}
