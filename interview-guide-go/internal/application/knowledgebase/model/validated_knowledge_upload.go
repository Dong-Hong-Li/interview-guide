package model

// ValidatedKnowledgeBaseUpload 由 controller 完成 HTTP 入参校验后封装，再传入 UploadKnowledgeBaseService。
// 正文抽取在 service 内通过 KnowledgeTextExtractor 完成。application/service 不处理 KBPostUploadRequest 等原始请求。
type ValidatedKnowledgeBaseUpload struct {
	Filename    string
	ContentType string
	Content     []byte
	Name        string
	Category    string
}
