package results

// UploadKnowledgeBaseResponse 知识库上传接口的响应体，含知识库元数据、对象存储引用与去重标记。
type UploadKnowledgeBaseResponse struct {
	KnowledgeBase UploadKBInfo    `json:"knowledgeBase"`
	Storage       UploadKBStorage `json:"storage"`
	Duplicate     bool            `json:"duplicate"`
}

// UploadKBInfo 知识库概要（新建或重复命中）。
type UploadKBInfo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	FileSize      int64  `json:"fileSize"`
	ContentLength int    `json:"contentLength"`
	VectorStatus  string `json:"vectorStatus"`
}

// UploadKBStorage 对象存储键与直链。
type UploadKBStorage struct {
	FileKey string `json:"fileKey"`
	FileURL string `json:"fileUrl"`
}
