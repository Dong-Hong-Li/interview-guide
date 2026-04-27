package resume

// InterviewerRole 与简历表 interviewer_role、上传入参、promptprofile 一致。
type InterviewerRole string

const (
	// InterviewerRoleBackend 后端 / 架构
	InterviewerRoleBackend InterviewerRole = "BACKEND"
	// InterviewerRoleFrontend 前端 / 客户端
	InterviewerRoleFrontend InterviewerRole = "FRONTEND"
)

// DefaultInterviewerRole 未传 interviewer_role 时的默认值（与 gorm 列 default 一致）。
const DefaultInterviewerRole = InterviewerRoleFrontend

// 与前端下拉的 label 展示一致。
const (
	InterviewerRoleBackendLabel  = "后端 / 架构"
	InterviewerRoleFrontendLabel = "前端 / 客户端"
)
