package controller

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/ragchat/model"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

// RagChatController RAG 对话 HTTP 适配层；当前全部端点固定返回 501（与主项目占位策略一致，实现后置）。
type RagChatController struct{}

// Register 将 /api/rag-chat/* 注册到 r。
func (c *RagChatController) Register(r chi.Router) {
	r.Route(APIMountPath, func(sr chi.Router) {
		sr.Post(PathSessions, binding.Handle(c.createSession))
		sr.Get(PathSessions, binding.Exec(c.listSessions))
		sr.Post(PathPostSessionMessagesStream, binding.Handle(c.sendMessageStream))
		sr.Get(PathGetSessionByID, binding.Handle(c.getSessionDetail))
		sr.Put(PathPutSessionTitle, binding.Handle(c.updateSessionTitle))
		sr.Put(PathPutSessionKnowledgeBases, binding.Handle(c.updateKnowledgeBases))
		sr.Put(PathPutSessionPin, binding.Handle(c.togglePin))
		sr.Delete(PathDeleteSession, binding.Handle(c.deleteSession))
	})
}

// createSession POST /api/rag-chat/sessions：创建 RAG 对话会话；当前 501 占位，实现后与主项目 rag-chat 对齐。
func (*RagChatController) createSession(_ context.Context, _ model.RagChatCreateSessionReq) (any, error) {
	return nil, notImplemented("ragChat.createSession")
}

// listSessions GET /api/rag-chat/sessions：会话列表（分页/排序以主产品为准）；当前 501 占位。
func (*RagChatController) listSessions(_ context.Context) (any, error) {
	return nil, notImplemented("ragChat.listSessions")
}

// sendMessageStream POST /api/rag-chat/sessions/{sessionId}/messages/stream：基于所选知识库流式发消息、SSE 出字；当前 501 占位。
func (*RagChatController) sendMessageStream(_ context.Context, _ model.RagChatSendMessageReq) (any, error) {
	return nil, notImplemented("ragChat.sendMessageStream")
}

// getSessionDetail GET /api/rag-chat/sessions/{sessionId}：会话详情与历史消息；当前 501 占位。
func (*RagChatController) getSessionDetail(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.getSessionDetail")
}

// updateSessionTitle PUT /api/rag-chat/sessions/{sessionId}/title：修改会话标题；当前 501 占位。
func (*RagChatController) updateSessionTitle(_ context.Context, _ model.RagChatUpdateTitleReq) (any, error) {
	return nil, notImplemented("ragChat.updateSessionTitle")
}

// updateKnowledgeBases PUT /api/rag-chat/sessions/{sessionId}/knowledge-bases：绑定/切换本会话引用的知识库；当前 501 占位。
func (*RagChatController) updateKnowledgeBases(_ context.Context, _ model.RagChatUpdateKnowledgeBasesReq) (any, error) {
	return nil, notImplemented("ragChat.updateKnowledgeBases")
}

// togglePin PUT /api/rag-chat/sessions/{sessionId}/pin：会话置顶/取消置顶；当前 501 占位。
func (*RagChatController) togglePin(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.togglePin")
}

// deleteSession DELETE /api/rag-chat/sessions/{sessionId}：删除会话及消息；当前 501 占位。
func (*RagChatController) deleteSession(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.deleteSession")
}

func notImplemented(h string) error {
	return response.Err(http.StatusNotImplemented, errmsg.NotImplemented+": "+h)
}
