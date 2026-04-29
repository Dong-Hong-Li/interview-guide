package errmsg

// RAG 对话（与 response.Err 搭配）
const (
	RagChatSessionServiceNil    = "RAG 对话会话服务未配置"
	RagChatSessionNotFound      = "RAG 会话不存在"
	RagChatTitleEmpty           = "标题不能为空"
	RagChatTitleTooLong         = "标题过长"
	RagChatInvalidKnowledgeBase = "存在无效的知识库编号"
	RagChatDeleteSuccess        = "RAG 会话已删除"
	RagChatUpdateTitleSuccess   = "标题已更新"
	RagChatUpdateKBsSuccess     = "知识库已更新"
	RagChatTogglePinSuccess     = "置顶状态已更新"
)
