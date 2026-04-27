package controller

const (
	// APIMountPath 简历域根路径，挂在统一 /api 之下。
	APIMountPath = "/resumes"
)

const (
	// PathUpload 上传简历
	PathUpload = "/upload"
	// PathInterviewerRoles 获取面试官角色
	PathInterviewerRoles = "/interviewer-roles"
	// PathStatistics 统计
	PathStatistics = "/statistics"
	// PathList 列表
	PathList = "/"
	// PathReanalyze 重新分析
	PathReanalyze = "/{id}/reanalyze"
	// PathDetail 详情
	PathDetail = "/{id}/detail"
	// PathExport 导出
	PathExport = "/{id}/export"
	// PathDelete 删除
	PathDelete = "/{id}"
)
