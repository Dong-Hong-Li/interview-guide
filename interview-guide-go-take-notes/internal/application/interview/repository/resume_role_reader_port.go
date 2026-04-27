package repository

import "context"

// InterviewerRoleReader 从简历表读取面试官人设（BACKEND/FRONTEND），供面试出题选 backend/frontend 模板。
type InterviewerRoleReader interface {
	InterviewerRoleByResumeID(ctx context.Context, resumeID int64) (string, error)
}
