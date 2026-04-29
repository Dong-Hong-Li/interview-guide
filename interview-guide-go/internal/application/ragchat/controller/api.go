// Package controller RAG 对话 API 路径片段，挂到 /api 下
package controller

const (
	// APIMountPath RAG 对话域根路径，挂在统一 /api 之下。
	APIMountPath = "/rag-chat"
)

// 相对已挂载在 /api 上的 chi Router 的 pattern（与主项目 ragchathttp.apis 一致）。
const (
	// PathSessions 会话列表 / 创建
	PathSessions = "/sessions"
	// PathPostSessionMessagesStream 流式发消息
	PathPostSessionMessagesStream = "/sessions/{sessionId}/messages/stream"
	// PathGetSessionByID 会话详情
	PathGetSessionByID = "/sessions/{sessionId}"
	// PathPutSessionTitle 更新标题
	PathPutSessionTitle = "/sessions/{sessionId}/title"
	// PathPutSessionKnowledgeBases 更新知识库
	PathPutSessionKnowledgeBases = "/sessions/{sessionId}/knowledge-bases"
	// PathPutSessionPin 置顶
	PathPutSessionPin = "/sessions/{sessionId}/pin"
	// PathDeleteSession 删除
	PathDeleteSession = "/sessions/{sessionId}"
)
