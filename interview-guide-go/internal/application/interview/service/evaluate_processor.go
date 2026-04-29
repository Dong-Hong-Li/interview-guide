package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	ivmodel "interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/repository"
	domainiv "interview-guide-go/internal/domain/interview"
)

// EvaluateProcessor 封装面试 LLM 评估 consumers 所需「抢单、拉题、写报告、刷缓存」能力（与主项目 interview.Service Worker* 对齐）。
type EvaluateProcessor struct {
	sessions repository.InterviewSessionWriter
	cache    repository.InterviewSessionCache
	resume   repository.ResumeTextSource
}

// NewEvaluateProcessor 由 cmd 组合根注入。
func NewEvaluateProcessor(
	sessions repository.InterviewSessionWriter,
	cache repository.InterviewSessionCache,
	resume repository.ResumeTextSource,
) *EvaluateProcessor {
	return &EvaluateProcessor{sessions: sessions, cache: cache, resume: resume}
}

// WorkerGetSessionByPublicID 供消费者按对外 sessionId 加载会话行。
func (p *EvaluateProcessor) WorkerGetSessionByPublicID(ctx context.Context, publicID string) (*ivmodel.WorkerSession, error) {
	if p == nil || p.sessions == nil {
		return nil, nil
	}
	return p.sessions.GetWorkerSessionByPublicID(ctx, strings.TrimSpace(publicID))
}

// WorkerTryMarkEvaluateProcessing PENDING → PROCESSING。
func (p *EvaluateProcessor) WorkerTryMarkEvaluateProcessing(ctx context.Context, sessionPK int64) (bool, error) {
	if p == nil || p.sessions == nil {
		return false, nil
	}
	return p.sessions.TryMarkInterviewSessionEvaluateProcessing(ctx, sessionPK)
}

// WorkerMarkEvaluateFailed 评估失败落库说明。
func (p *EvaluateProcessor) WorkerMarkEvaluateFailed(ctx context.Context, sessionPK int64, errMsg string) error {
	if p == nil || p.sessions == nil {
		return nil
	}
	return p.sessions.MarkInterviewSessionEvaluateFailed(ctx, sessionPK, errMsg)
}

// WorkerListInterviewAnswers 拉取已提交答案，与 questions_json 按题号合并。
func (p *EvaluateProcessor) WorkerListInterviewAnswers(ctx context.Context, sessionPK int64) ([]ivmodel.WorkerAnswer, error) {
	if p == nil || p.sessions == nil {
		return nil, nil
	}
	return p.sessions.ListInterviewAnswersBySessionPK(ctx, sessionPK)
}

// WorkerSaveEvaluationResult 将 LLM 报告写入 DB。
func (p *EvaluateProcessor) WorkerSaveEvaluationResult(ctx context.Context, sessionPK int64, report *ivmodel.EvaluationReport) error {
	if p == nil || p.sessions == nil {
		return nil
	}
	return p.sessions.SaveInterviewEvaluationResult(ctx, sessionPK, report)
}

// WorkerGetResumeTextAndInterviewerRole 与主项目 WorkerGetResumeRow 等价的简历正文 + 角色。
func (p *EvaluateProcessor) WorkerGetResumeTextAndInterviewerRole(ctx context.Context, resumeID int64) (resumeText, interviewerRole string, err error) {
	if p == nil || p.resume == nil {
		return "", "", errors.New("resume source not configured")
	}
	return p.resume.ResumeTextAndInterviewerRole(ctx, resumeID)
}

// WorkerWarmSessionCacheAfterEvaluate 评估成功后将 EVALUATED 态 + 题面写回 Redis。
func (p *EvaluateProcessor) WorkerWarmSessionCacheAfterEvaluate(ctx context.Context, publicID string) {
	if p == nil || p.cache == nil || p.sessions == nil {
		return
	}
	sid := strings.TrimSpace(publicID)
	if sid == "" {
		return
	}
	sess, err := p.sessions.GetSessionBySessionID(ctx, sid)
	if err != nil || sess == nil {
		return
	}
	display := ""
	if p.resume != nil {
		if t, _, rerr := p.resume.ResumeTextAndInterviewerRole(ctx, sess.ResumeID); rerr == nil {
			display = t
		}
	}
	qjs2, err2 := json.Marshal(sess.Questions)
	if err2 != nil {
		return
	}
	adv := len(sess.Questions)
	if sess.TotalQuestions > 0 {
		adv = sess.TotalQuestions
	}
	rid := sess.ResumeID
	st := strings.TrimSpace(string(sess.Status))
	if st == "" {
		st = domainiv.InterviewStatusEvaluated
	}
	at := adv
	_ = p.cache.SaveSession(ctx, sess.SessionID, display, &rid, string(qjs2), sess.CurrentQuestionIndex, st, &at)
}
