package model

// InterviewerRoleOption 面试官角色下拉项（与上传简历可选值一致，供 GET /interviewer-roles 使用）。
type InterviewerRoleOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
