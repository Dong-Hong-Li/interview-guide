package controller

import (
	"context"
	"interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/service"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type InterviewController struct {
	CreateInterviewService       *service.CreateInterviewService
	UnfinishedSessionService     *service.UnfinishedSessionService
	CurrentQuestionService       *service.CurrentQuestionService
	SubmitAnswerService          *service.SubmitAnswerService
	ListInterviewSessionsService *service.ListInterviewSessionsService
	ReportService                *service.ReportService
	GetInterviewDetailService    *service.GetInterviewDetailService
	GetSessionService            *service.GetSessionService
	CompleteSessionService       *service.CompleteSessionService
	DeleteInterviewService       *service.DeleteInterviewService
	Logger                       *zap.Logger
}

func (c *InterviewController) Register(r chi.Router) {
	r.Route(APIMountPath, func(sr chi.Router) {
		sr.Post(InterviewNewSessions, binding.Handle(c.createInterview))
		sr.Get(InterviewNewSessions, binding.Handle(c.listInterviewSessions))
		sr.Get(PathGetUnfinishedByResume, binding.Handle(c.findUnfinishedInterview))
		sr.Get(PathGetSessionQuestion, binding.Handle(c.getCurrentQuestion))
		sr.Get(PathGetSessionReport, binding.Handle(c.getReport))
		sr.Get(PathGetSessionDetails, binding.Handle(c.getInterviewDetail))
		sr.Get(PathGetSessionExport, c.handleExportInterviewPDF)
		sr.Delete(PathDeleteSession, binding.Handle(c.deleteInterview))
		sr.Post(PathSessionAnswers, binding.Handle(c.submitAnswer))
		sr.Put(PathSessionAnswers, binding.Handle(c.saveAnswer))
		sr.Post(PathPostSessionComplete, binding.Handle(c.completeInterview))
		sr.Get(PathGetSession, binding.Handle(c.getSession))
	})
}

// createIntervie
// 题目生成、Redis/DB 持久化接入后在此组装返回。
func (c *InterviewController) createInterview(ctx context.Context, in model.CreateInterviewSessionReq) (*results.InterviewSession, error) {
	if strings.TrimSpace(in.ResumeText) == "" {
		return nil, response.Err(http.StatusBadRequest, "resumeText is required")
	}
	if in.QuestionCount < 3 || in.QuestionCount > 20 {
		return nil, response.Err(http.StatusBadRequest, "questionCount must be between 3 and 20")
	}
	if in.ResumeID == nil {
		return nil, response.Err(http.StatusBadRequest, "resumeId is required")
	}

	out, err := c.CreateInterviewService.CreateInterview(ctx, model.ValidatedCreateInterviewSession{
		ResumeText:    strings.TrimSpace(in.ResumeText),
		QuestionCount: in.QuestionCount,
		ResumeID:      *in.ResumeID,
		ForceCreate:   in.ForceCreate,
	})
	if err != nil {
		c.Logger.Error("createInterview failed", zap.Error(err))
		return nil, err
	}
	return out, nil
}

// listInterviewSessions GET /api/interview/sessions?page=&size=
func (c *InterviewController) listInterviewSessions(ctx context.Context, in model.ListInterviewSessionsReq) (*results.InterviewListPage, error) {
	if c.ListInterviewSessionsService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "list interview sessions not configured")
	}
	size := in.Size
	if size == 0 {
		size = in.PageSize
	}
	return c.ListInterviewSessionsService.List(ctx, in.Page, size)
}

// findUnfinishedInterview GET /api/interview/sessions/unfinished/{resumeId}
func (c *InterviewController) findUnfinishedInterview(ctx context.Context, in model.FindUnfinishedReq) (*results.InterviewSession, error) {
	if in.ResumeID < 1 {
		return nil, response.Err(http.StatusBadRequest, errmsg.FindUnfinishedBadResumeID)
	}
	return c.UnfinishedSessionService.FindUnfinishedSession(ctx, in.ResumeID)
}

// getCurrentQuestion GET /api/interview/sessions/{sessionId}/question
func (c *InterviewController) getCurrentQuestion(ctx context.Context, in model.GetCurrentQuestionReq) (*results.CurrentQuestionResponse, error) {
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, "sessionId is required")
	}
	if c.CurrentQuestionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "current question not configured")
	}
	return c.CurrentQuestionService.GetCurrentQuestion(ctx, sid)
}

// deleteInterview DELETE /api/interview/sessions/{sessionId}
func (c *InterviewController) deleteInterview(ctx context.Context, in model.DeleteInterviewSessionReq) (map[string]string, error) {
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID)
	}
	return c.DeleteInterviewService.DeleteSession(ctx, sid)
}

// submitAnswer POST /api/interview/sessions/{sessionId}/answers（单题，body: questionIndex + answer，与前端一致）
func (c *InterviewController) submitAnswer(ctx context.Context, in model.SubmitAnswerReq) (*results.SubmitAnswerResponse, error) {
	if c.SubmitAnswerService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "submit answer not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, "sessionId is required")
	}
	answer := strings.TrimSpace(in.Answer)
	if answer == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerAnswerEmpty)
	}
	if in.QuestionIndex < 0 {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerBadQuestionIndex)
	}
	return c.SubmitAnswerService.SubmitAnswer(ctx, model.ValidatedSubmitAnswer{
		SessionID:     sid,
		QuestionIndex: in.QuestionIndex,
		Answer:        answer,
	})
}

// saveAnswer PUT /api/interview/sessions/{sessionId}/answers：仅保存草稿，不推进游标、不入队评估。
func (c *InterviewController) saveAnswer(ctx context.Context, in model.SaveAnswerReq) (*results.SubmitAnswerResponse, error) {
	if c.SubmitAnswerService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "submit answer not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, "sessionId is required")
	}
	if in.QuestionIndex < 0 {
		return nil, response.Err(http.StatusBadRequest, errmsg.SubmitAnswerBadQuestionIndex)
	}
	answer := strings.TrimSpace(in.Answer)
	return c.SubmitAnswerService.SaveAnswer(ctx, model.ValidatedSaveAnswer{
		SessionID:     sid,
		QuestionIndex: in.QuestionIndex,
		Answer:        answer,
	})
}

// getReport GET /api/interview/sessions/{sessionId}/report 查询面试报告
func (c *InterviewController) getReport(ctx context.Context, in model.GetReportReq) (*results.InterviewReport, error) {
	if c.ReportService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview report not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, "sessionId is required")
	}
	return c.ReportService.GetReport(ctx, sid)
}

// getInterviewDetail GET /api/interview/sessions/{sessionId}/details
func (c *InterviewController) getInterviewDetail(ctx context.Context, in model.GetInterviewDetailReq) (*results.InterviewDetail, error) {
	if c.GetInterviewDetailService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview details not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID)
	}
	return c.GetInterviewDetailService.GetDetail(ctx, sid)
}

// handleExportInterviewPDF GET /api/interview/sessions/{sessionId}/export，body 为 application/pdf 二进制（与主项目一致，不用 JSON binding）。
func (c *InterviewController) handleExportInterviewPDF(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if c.ReportService == nil {
		response.WriteErr(w, response.Err(http.StatusServiceUnavailable, "interview export not configured"))
		return
	}
	if sid == "" {
		response.WriteErr(w, response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID))
		return
	}
	out, disp, err := c.ReportService.ExportInterviewPDF(r.Context(), sid)
	if err != nil {
		response.WriteErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", disp)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

// getSession GET /api/interview/sessions/{sessionId} 答题页/详情轮询，看当前做到第几题、题面、已答内容、会话状态等。。
func (c *InterviewController) getSession(ctx context.Context, in model.GetSessionReq) (*results.InterviewSession, error) {
	if c.GetSessionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "get session not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID)
	}
	return c.GetSessionService.GetSession(ctx, sid)
}

// completeInterview POST /api/interview/sessions/{sessionId}/complete 提前交卷并触发评估入队（与主项目 complete 对齐）。
func (c *InterviewController) completeInterview(ctx context.Context, in model.CompleteSessionReq) (any, error) {
	if c.CompleteSessionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "complete session not configured")
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.DeleteInterviewSessionBadID)
	}

	return c.CompleteSessionService.CompleteSession(ctx, sid)
}
