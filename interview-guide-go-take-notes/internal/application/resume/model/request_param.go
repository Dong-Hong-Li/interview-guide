package model

// ListResumeReq 列表分页查询参数（由 HTTP binding 按 query 标签注入）。
// GET /api/resumes/?page=&size=&pageSize=；Size 为 0 时控制器可回退为 PageSize。
type ListResumeRequest struct {
	Page     int `query:"page"`
	Size     int `query:"size"`
	PageSize int `query:"pageSize"`
}

// UploadResumeRequest 上传简历请求参数
// POST /api/resumes/upload — multipart/form-data 时由 binding 按 form 标签填充；JSON 时按 json 标签（测试或工具用）。
// 文件字段名与前端约定为 file。Filename/ContentType 在 multipart 下由 file 的 FileHeader 自动写入，勿手填。
type UploadResumeRequest struct {
	Filename        string `json:"filename" form:"-"`
	ContentType     string `json:"content_type" form:"-"`
	Content         []byte `json:"content" form:"file" validate:"required"`
	InterviewerRole string `json:"interviewer_role" form:"interviewerRole"`
}

// IDPathRequest 主键路径参数（由 HTTP binding 按 path 标签注入）。
// DELETE /api/resumes/{id}
type IDPathRequest struct {
	ID int64 `path:"id"`
}
