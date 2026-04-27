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

func (*RagChatController) createSession(_ context.Context, _ model.RagChatCreateSessionReq) (any, error) {
	return nil, notImplemented("ragChat.createSession")
}
func (*RagChatController) listSessions(_ context.Context) (any, error) {
	return nil, notImplemented("ragChat.listSessions")
}
func (*RagChatController) sendMessageStream(_ context.Context, _ model.RagChatSendMessageReq) (any, error) {
	return nil, notImplemented("ragChat.sendMessageStream")
}
func (*RagChatController) getSessionDetail(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.getSessionDetail")
}
func (*RagChatController) updateSessionTitle(_ context.Context, _ model.RagChatUpdateTitleReq) (any, error) {
	return nil, notImplemented("ragChat.updateSessionTitle")
}
func (*RagChatController) updateKnowledgeBases(_ context.Context, _ model.RagChatUpdateKnowledgeBasesReq) (any, error) {
	return nil, notImplemented("ragChat.updateKnowledgeBases")
}
func (*RagChatController) togglePin(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.togglePin")
}
func (*RagChatController) deleteSession(_ context.Context, _ model.RagChatSessionPathReq) (any, error) {
	return nil, notImplemented("ragChat.deleteSession")
}

func notImplemented(h string) error {
	return response.Err(http.StatusNotImplemented, errmsg.NotImplemented+": "+h)
}
