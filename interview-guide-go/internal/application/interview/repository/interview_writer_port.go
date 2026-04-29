package repository

import (
	"context"

	ivmodel "interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/model/results"
)

// InterviewSessionWriter 面试会话持久化端口（GORM/Mapper 等实现落库、历史题拉取、未完成会话查询）。
type InterviewSessionWriter interface {
	// InsertInterviewSession 持久化新会话（含 questions）；入参为完整 API 形态，由实现写 Redis/库。
	InsertInterviewSession(ctx context.Context, session *results.InterviewSession) error
	// 查找未完成的面试会话
	FindUnfinishedSession(ctx context.Context, resumeID int64) (*results.InterviewSession, error)
	// 拉历史题目（去重/差异化出题）
	GetHistoricalQuestionsByResumeID(ctx context.Context, resumeID int64) ([]string, error)
	// GetSessionBySessionID 按对外 sessionId 加载完整会话（含 questions_json 与简历正文）。
	GetSessionBySessionID(ctx context.Context, sessionID string) (*results.InterviewSession, error)
	// DeleteInterviewSessionByPublicID 按对外 session_id 删除本会话及其 interview_answers 行；无则返回 gorm.ErrRecordNotFound。
	DeleteInterviewSessionByPublicID(ctx context.Context, publicSessionID string) error

	// GetSessionRecordForSubmit 按对外 sessionId 加载行快照（含内部主键），无行时 (nil, nil)。
	GetSessionRecordForSubmit(ctx context.Context, sessionID string) (*results.SessionRecordForSubmit, error)
	// SaveInterviewAnswer 写入或更新 interview_answers（session_id 为父表主键）。
	SaveInterviewAnswer(ctx context.Context, sessionPK int64, qIdx int, question, category, userAnswer string, score *int, feedback string) error
	// UpdateInterviewSessionProgress 更新当前题下标与会话状态；setCompleted 时写 completed_at。
	UpdateInterviewSessionProgress(ctx context.Context, sessionPK int64, currentIdx int, status string, setCompleted bool) error

	// LoadForReport 按对外 sessionId 查会话行与全部答题，供报告接口。
	LoadForReport(ctx context.Context, publicSessionID string) (*results.SessionReportDB, []results.InterviewAnswerDB, error)

	// ── 面试 LLM 评估（最后一题交卷后 PENDING + 入队，消费者与落库与主项目一致）──
	UpdateInterviewSessionEvaluatePending(ctx context.Context, sessionPK int64) error
	// GetWorkerSessionByPublicID 按对外 sessionId 加载会话行（含内部主键），无行时 (nil, nil)。
	GetWorkerSessionByPublicID(ctx context.Context, publicSessionID string) (*ivmodel.WorkerSession, error)
	// TryMarkInterviewSessionEvaluateProcessing 尝试标记评估状态为处理中
	TryMarkInterviewSessionEvaluateProcessing(ctx context.Context, sessionPK int64) (bool, error)
	// MarkInterviewSessionEvaluateFailed 标记评估失败
	MarkInterviewSessionEvaluateFailed(ctx context.Context, sessionPK int64, errMsg string) error
	// SaveInterviewEvaluationResult 保存评估结果
	SaveInterviewEvaluationResult(ctx context.Context, sessionPK int64, report *ivmodel.EvaluationReport) error
	// ListInterviewAnswersBySessionPK 按内部主键列出答题
	ListInterviewAnswersBySessionPK(ctx context.Context, sessionPK int64) ([]ivmodel.WorkerAnswer, error)

	// ListInterviewSessionsPage 全库面试会话分页，按 created_at 降序；行内带 resume 文件名（与前端 InterviewListPage 一致）。
	ListInterviewSessionsPage(ctx context.Context, page, size int) ([]results.InterviewListItem, int64, error)
}
