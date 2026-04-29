package repository

import (
	"context"
	"time"
)

// InterviewSessionCache 面试会话缓存
type InterviewSessionCache interface {
	// 保存会话
	SaveSession(ctx context.Context, sessionID, resumeText string, resumeID *int64, questionsJSON string, currentIndex int, status string, advertisedTotalQuestions *int) error

	// DeleteSessionKeys 删除 interview:session:* 与 resumeId→sessionId 索引，避免会话删除后留下脏缓存。
	DeleteSessionKeys(ctx context.Context, sessionID string, resumeID int64) error

	// TryAcquireCreatingLock 同简历并发创建会话时，仅一路应调用 LLM；在出题落库前用 Redis SETNX 互斥。成功为 acquired=true。
	// 对应键过期时间与出题总超时一致，进程崩溃可自动恢复。
	TryAcquireCreatingLock(ctx context.Context, resumeID int64, lockTTL time.Duration) (acquired bool, err error)
	// ReleaseCreatingLock 出题流程结束（成功或失败）后释放，使后续请求可再创建。
	ReleaseCreatingLock(ctx context.Context, resumeID int64) error
}
