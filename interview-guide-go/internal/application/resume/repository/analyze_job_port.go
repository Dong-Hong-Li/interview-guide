package repository

import (
	"context"
)

// AnalyzeJob 交给消费者用的纯业务载荷。
type AnalyzeJob struct {
	ResumeID   int64
	ResumeText string
	// 可选：InterviewerRole、RequestID、Attempt 等
}

// AnalyzePublisher 简历分析队列端口。
type AnalyzePublisher interface {
	// SendAnalyzeTask 发送简历分析任务到队列。
	SendAnalyzeTask(ctx context.Context, resumeID int64, resumeText string) error
}
