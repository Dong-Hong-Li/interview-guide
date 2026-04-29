package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	"interview-guide-go/shared/interview"
	"interview-guide-go/shared/response"
	"interview-guide-go/shared/uuid"
)

type CreateInterviewService struct {
	interviewWriter            repository.InterviewSessionWriter
	interviewQuestionGenerator repository.InterviewQuestionGenerator
	interviewSessionCache      repository.InterviewSessionCache
	interviewerRoleReader      repository.InterviewerRoleReader
}

// createInterviewWorkMax 出题+落库可能远长于浏览器/反代对「单次请求」的等待（常见 ~60～90s），
// 若全程使用 *http.Request 的 ctx，断连时会产生 context canceled。与请求解绑后仍需上限以防泄漏。
const createInterviewWorkMax = 10 * time.Minute

// NewCreateInterviewService 由 wire 或 main 注入依赖。
func NewCreateInterviewService(
	w repository.InterviewSessionWriter,
	gen repository.InterviewQuestionGenerator,
	cache repository.InterviewSessionCache,
	roleReader repository.InterviewerRoleReader,
) *CreateInterviewService {
	return &CreateInterviewService{
		interviewWriter:            w,
		interviewQuestionGenerator: gen,
		interviewSessionCache:      cache,
		interviewerRoleReader:      roleReader,
	}
}

// CreateInterview 创建面试会话：抢 Redis 创建锁、异步生成题目、状态置 QUESTIONS_PENDING；入参须为 controller 校验后的 ValidatedCreateInterviewSession。
func (s *CreateInterviewService) CreateInterview(ctx context.Context, in model.ValidatedCreateInterviewSession) (*results.InterviewSession, error) {
	if s.interviewWriter == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview writer not configured")
	}
	if s.interviewQuestionGenerator == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interview question generator not configured")
	}
	if s.interviewerRoleReader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, "interviewer role reader not configured")
	}

	// 与 *http.Request 的取消解绑，避免仅因客户端/代理在 ~90s 弃等导致 LLM 与落库被 context canceled。
	// 仍保留服务侧总超时 createInterviewWorkMax。
	workCtx, workCancel := context.WithTimeout(context.WithoutCancel(ctx), createInterviewWorkMax)
	defer workCancel()

	resumeID := in.ResumeID

	if !in.ForceCreate {
		unfinished, err := s.interviewWriter.FindUnfinishedSession(workCtx, resumeID)
		if err != nil {
			return nil, err
		}
		if unfinished != nil {
			return unfinished, nil
		}
	}

	// 未终态会话仅在 Postgres 有行时可见；在 LLM 完成并 Insert 前并发多次创建会通过「都查不到」而重复打模型。
	// Redis SETNX 互斥（SaveSession 之前不会写入 Redis），与 createInterviewWorkMax 同 TTL。
	lockHeld := false
	if s.interviewSessionCache != nil {
		acquired, aerr := s.interviewSessionCache.TryAcquireCreatingLock(workCtx, resumeID, createInterviewWorkMax)
		if aerr != nil {
			return nil, aerr
		}
		if !acquired {
			if !in.ForceCreate {
				retry, rerr := s.interviewWriter.FindUnfinishedSession(workCtx, resumeID)
				if rerr != nil {
					return nil, rerr
				}
				if retry != nil {
					return retry, nil
				}
			}
			return nil, response.Err(http.StatusConflict, "该简历的面试题正在生成中，请稍后再试")
		}
		lockHeld = true
	}
	if lockHeld {
		defer func() {
			relCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = s.interviewSessionCache.ReleaseCreatingLock(relCtx, resumeID)
		}()
	}

	sessionID := uuid.NewUUID16()

	historical, err := s.interviewWriter.GetHistoricalQuestionsByResumeID(workCtx, resumeID)
	if err != nil {
		return nil, err
	}
	// 获取面试官角色
	interviewerRole, err := s.interviewerRoleReader.InterviewerRoleByResumeID(workCtx, resumeID)
	if err != nil {
		return nil, err
	}

	// 生成面试题列表
	questions, err := s.interviewQuestionGenerator.GenerateQuestions(workCtx, in.ResumeText, in.QuestionCount, historical, interviewerRole)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, response.Err(http.StatusBadGateway, "no questions generated")
	}

	out := &results.InterviewSession{
		SessionID:            sessionID,
		ResumeID:             resumeID,
		ResumeText:           in.ResumeText,
		TotalQuestions:       len(questions),
		CurrentQuestionIndex: 0,
		Questions:            questions,
		Status:               interview.StatusCreated,
	}

	questionsJSONBytes, mErr := json.Marshal(questions)
	if mErr != nil {
		return nil, mErr
	}
	tq := len(questions)
	statusStr := string(interview.StatusCreated)
	if s.interviewSessionCache != nil {
		if cErr := s.interviewSessionCache.SaveSession(workCtx, sessionID, in.ResumeText, &resumeID,
			string(questionsJSONBytes), 0, statusStr, &tq); cErr != nil {
			return nil, cErr
		}
	}
	if err := s.interviewWriter.InsertInterviewSession(workCtx, out); err != nil {
		return nil, err
	}
	return out, nil
}
