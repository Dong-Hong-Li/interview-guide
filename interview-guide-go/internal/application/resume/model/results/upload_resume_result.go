package result

// UploadStorage 与 Java / interview-guide 前端约定的 storage 块（UploadPage 读取 data.storage.resumeId）。
type UploadStorage struct {
	FileKey  string `json:"fileKey"`
	FileURL  string `json:"fileUrl"`
	ResumeID int64  `json:"resumeId"` // 须为 JSON 数字且在 JS 安全整数内；落库后改为 DB 主键
}

// UploadResumeResult POST /api/resumes/upload 成功体（与 interview-guide-frontend/src/pages/UploadPage.tsx 对齐）。
type UploadResumeResult struct {
	Storage   UploadStorage `json:"storage"`
	Duplicate bool          `json:"duplicate,omitempty"`
}
