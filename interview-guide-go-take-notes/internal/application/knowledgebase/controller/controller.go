package controller

import (
	"context"
	"net/http"

	"interview-guide-go/internal/application/knowledgebase/model"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

// KnowledgeBaseController 知识库 HTTP 适配层；当前全部端点固定返回 501（与主项目占位策略一致，实现后置）。
type KnowledgeBaseController struct{}

// Register 将 /api/knowledgebase/* 注册到 r。
func (c *KnowledgeBaseController) Register(r chi.Router) {
	r.Route(APIMountPath, func(sr chi.Router) {
		sr.Post(PathPostUpload, binding.Handle(c.uploadKnowledgeBase))
		sr.Get(PathGetList, binding.Exec(c.getAllKnowledgeBases))
		sr.Get(PathGetCategories, binding.Exec(c.getAllCategories))
		sr.Get(PathGetByCategory, binding.Handle(c.getByCategory))
		sr.Get(PathGetUncategorized, binding.Exec(c.getUncategorized))
		sr.Get(PathGetSearch, binding.Handle(c.search))
		sr.Get(PathGetStats, binding.Exec(c.getStatistics))
		sr.Post(PathPostQueryStream, binding.Handle(c.queryKnowledgeBaseStream))
		sr.Post(PathPostQuery, binding.Handle(c.queryKnowledgeBase))
		sr.Get(PathGetByIDDownload, binding.Handle(c.downloadKnowledgeBase))
		sr.Get(PathByID, binding.Handle(c.getKnowledgeBase))
		sr.Delete(PathByID, binding.Handle(c.deleteKnowledgeBase))
		sr.Put(PathPutByIDCategory, binding.Handle(c.updateCategory))
		sr.Post(PathPostByIDRevectorize, binding.Handle(c.revectorize))
	})
}

func (*KnowledgeBaseController) getAllKnowledgeBases(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getAllKnowledgeBases")
}
func (*KnowledgeBaseController) getAllCategories(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getAllCategories")
}
func (*KnowledgeBaseController) getUncategorized(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getUncategorized")
}
func (*KnowledgeBaseController) getStatistics(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getStatistics")
}
func (*KnowledgeBaseController) uploadKnowledgeBase(_ context.Context, _ model.KBPostUploadNoBody) (any, error) {
	return nil, notImplemented("knowledgebase.uploadKnowledgeBase")
}
func (*KnowledgeBaseController) getByCategory(_ context.Context, _ model.KBCategoryPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.getByCategory")
}
func (*KnowledgeBaseController) search(_ context.Context, _ model.KBSearchReq) (any, error) {
	return nil, notImplemented("knowledgebase.search")
}
func (*KnowledgeBaseController) queryKnowledgeBaseStream(_ context.Context, _ model.KBQueryReq) (any, error) {
	return nil, notImplemented("knowledgebase.queryKnowledgeBaseStream")
}
func (*KnowledgeBaseController) queryKnowledgeBase(_ context.Context, _ model.KBQueryReq) (any, error) {
	return nil, notImplemented("knowledgebase.queryKnowledgeBase")
}
func (*KnowledgeBaseController) downloadKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.downloadKnowledgeBase")
}
func (*KnowledgeBaseController) getKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.getKnowledgeBase")
}
func (*KnowledgeBaseController) deleteKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.deleteKnowledgeBase")
}
func (*KnowledgeBaseController) updateCategory(_ context.Context, _ model.KBUpdateCategoryReq) (any, error) {
	return nil, notImplemented("knowledgebase.updateCategory")
}
func (*KnowledgeBaseController) revectorize(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.revectorize")
}

func notImplemented(h string) error {
	return response.Err(http.StatusNotImplemented, errmsg.NotImplemented+": "+h)
}
