package controller

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/ragchat/model"
	"interview-guide-go/internal/application/ragchat/service"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

// RagChatController RAG 对话 HTTP 适配层；流式发消息仍为 501。
type RagChatController struct {
	SessionService *service.RagChatSessionService
}

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

// createSession POST /api/rag-chat/sessions
func (c *RagChatController) createSession(ctx context.Context, req model.RagChatCreateSessionReq) (any, error) {
	if c == nil || c.SessionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return nil, err
	}
	return c.SessionService.Create(ctx, req.Title, req.KnowledgeBaseIds)
}

// listSessions GET /api/rag-chat/sessions
func (c *RagChatController) listSessions(ctx context.Context) (any, error) {
	if c == nil || c.SessionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	return c.SessionService.List(ctx)
}

// sendMessageStream POST /api/rag-chat/sessions/{sessionId}/messages/stream
func (*RagChatController) sendMessageStream(_ context.Context, _ model.RagChatSendMessageReq) (any, error) {
	return nil, notImplemented("ragChat.sendMessageStream")
}

// getSessionDetail GET /api/rag-chat/sessions/{sessionId}
func (c *RagChatController) getSessionDetail(ctx context.Context, req model.RagChatSessionPathReq) (any, error) {
	if c == nil || c.SessionService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return nil, err
	}
	if req.SessionID < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid session id")
	}
	return c.SessionService.GetDetail(ctx, req.SessionID)
}

// updateSessionTitle PUT /api/rag-chat/sessions/{sessionId}/title
func (c *RagChatController) updateSessionTitle(ctx context.Context, req model.RagChatUpdateTitleReq) (string, error) {
	if c == nil || c.SessionService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return "", err
	}
	if req.SessionID < 1 {
		return "", response.Err(http.StatusBadRequest, "invalid session id")
	}
	if err := c.SessionService.UpdateTitle(ctx, req.SessionID, req.Title); err != nil {
		return "", err
	}
	return errmsg.RagChatUpdateTitleSuccess, nil
}

// updateKnowledgeBases PUT /api/rag-chat/sessions/{sessionId}/knowledge-bases
func (c *RagChatController) updateKnowledgeBases(ctx context.Context, req model.RagChatUpdateKnowledgeBasesReq) (string, error) {
	if c == nil || c.SessionService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return "", err
	}
	if req.SessionID < 1 {
		return "", response.Err(http.StatusBadRequest, "invalid session id")
	}
	if err := c.SessionService.UpdateKnowledgeBases(ctx, req.SessionID, req.KnowledgeBaseIds); err != nil {
		return "", err
	}
	return errmsg.RagChatUpdateKBsSuccess, nil
}

// togglePin PUT /api/rag-chat/sessions/{sessionId}/pin
func (c *RagChatController) togglePin(ctx context.Context, req model.RagChatSessionPathReq) (string, error) {
	if c == nil || c.SessionService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return "", err
	}
	if req.SessionID < 1 {
		return "", response.Err(http.StatusBadRequest, "invalid session id")
	}
	if err := c.SessionService.TogglePin(ctx, req.SessionID); err != nil {
		return "", err
	}
	return errmsg.RagChatTogglePinSuccess, nil
}

// deleteSession DELETE /api/rag-chat/sessions/{sessionId}
func (c *RagChatController) deleteSession(ctx context.Context, req model.RagChatSessionPathReq) (string, error) {
	if c == nil || c.SessionService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.RagChatSessionServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return "", err
	}
	if req.SessionID < 1 {
		return "", response.Err(http.StatusBadRequest, "invalid session id")
	}
	if err := c.SessionService.Delete(ctx, req.SessionID); err != nil {
		return "", err
	}
	return errmsg.RagChatDeleteSuccess, nil
}

func notImplemented(h string) error {
	return response.Err(http.StatusNotImplemented, errmsg.NotImplemented+": "+h)
}
