package repository

import "context"

// ResumeTextSource 为面试评估消费者提供 resume_text 与面试官角色（与 WorkerGetResumeRow 语义一致）。
type ResumeTextSource interface {
	ResumeTextAndInterviewerRole(ctx context.Context, resumeID int64) (resumeText, interviewerRole string, err error)
}

// InterviewEvaluateEnqueuer 投递「按会话 LLM 评估」Redis Stream 任务；无 Redis 时可注入 no-op 实现。
type InterviewEvaluateEnqueuer interface {
	EnqueueInterviewEvaluate(ctx context.Context, sessionPublicID string) error
}
