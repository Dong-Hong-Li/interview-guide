package model

// RagChatSessionPathReq 通过路径 {sessionId} 标识 RAG 对话会话。
type RagChatSessionPathReq struct {
	SessionID string `path:"sessionId"`
}

// RagChatCreateSessionReq 新建 RAG 对话会话（JSON body）。
// POST /api/rag-chat/sessions
type RagChatCreateSessionReq struct {
	KnowledgeBaseIds []int64 `json:"knowledgeBaseIds"`
	Title            string  `json:"title,omitempty"`
}

// RagChatSendMessageReq 发送消息（JSON body + path）。
// POST /api/rag-chat/sessions/{sessionId}/messages/stream
type RagChatSendMessageReq struct {
	SessionID string `path:"sessionId" json:"-"`
	Question  string `json:"question"`
}

// RagChatUpdateTitleReq 更新会话标题（JSON body + path）。
// PUT /api/rag-chat/sessions/{sessionId}/title
type RagChatUpdateTitleReq struct {
	SessionID string `path:"sessionId" json:"-"`
	Title     string `json:"title"`
}

// RagChatUpdateKnowledgeBasesReq 更新会话关联知识库（JSON body + path）。
// PUT /api/rag-chat/sessions/{sessionId}/knowledge-bases
type RagChatUpdateKnowledgeBasesReq struct {
	SessionID        string  `path:"sessionId" json:"-"`
	KnowledgeBaseIds []int64 `json:"knowledgeBaseIds"`
}
