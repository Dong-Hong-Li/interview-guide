package errmsg

// 知识库上传与持久化（与 response.Err 搭配）
const (
	KnowledgeBaseUploadServiceNil           = "knowledge base upload service is nil"
	KnowledgeBaseWriterNotConfigured        = "knowledge base persistence not configured"
	KnowledgeBaseTextExtractorNotConfigured = "knowledge base text extractor not configured"
	KnowledgeBaseExtractTextEmpty           = "无法从文件中提取文本内容，请确保文件格式正确"
	KnowledgeBaseVectorizeChunkEmpty        = "向量化分块后为空"
)

// 与 err.Error() 拼接的固定前缀
const (
	FindKnowledgeBaseByHashFailed = "find knowledge base by hash failed: "
	UploadKnowledgeBaseFileFailed = "upload knowledge base file failed: "
	GetKnowledgeBaseURLFailed     = "get knowledge base object url failed: "
	SaveKnowledgeBaseFailed       = "save knowledge base failed: "
	SendVectorizeTaskFailed       = "send vectorize task failed: "
)
