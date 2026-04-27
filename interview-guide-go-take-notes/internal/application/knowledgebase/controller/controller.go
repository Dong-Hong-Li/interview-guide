package controller

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model"
	"interview-guide-go/internal/application/knowledgebase/service"
	domainkb "interview-guide-go/internal/domain/knowledgebase"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

// KnowledgeBaseController 知识库 HTTP 适配层；upload 已实现，其余端点多为 501 占位。
type KnowledgeBaseController struct {
	UploadService *service.UploadKnowledgeBaseService
}

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

// uploadKnowledgeBase POST /api/knowledgebase/upload：上传知识库文件，解析、落库与入队向量化以主产品为准；
// 与 Java example/modules/knowledgebase 对齐时建议按序落地（可拆为 UploadService + 复用 storage/mapper/Redis，参考 Resume 上传与面试评估消费者形态）：
func (c *KnowledgeBaseController) uploadKnowledgeBase(ctx context.Context, request model.KBPostUploadRequest) (any, error) {
	if c == nil || c.UploadService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseUploadServiceNil)
	}
	// 1. 绑定 multipart：使用 model.KBPostUploadRequest（file、name、category）
	if err := binding.Validate(&request); err != nil {
		return nil, err
	}

	filename := strings.TrimSpace(request.Filename)
	if filename == "" {
		return nil, response.Err(http.StatusBadRequest, "filename is required")
	}
	contentType := strings.TrimSpace(request.ContentType)
	if contentType == "" {
		return nil, response.Err(http.StatusBadRequest, "content type is required")
	}
	content := request.Content
	if len(content) == 0 {
		return nil, response.Err(http.StatusBadRequest, "content is required")
	}
	// 2. 类型：detectContentType + 白名单（不支持的类型直接 400）
	if err := domainkb.ValidateContentType(contentType); err != nil {
		return nil, response.Err(http.StatusBadRequest, err.Error())
	}
	fh := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(content)),
		Header: textproto.MIMEHeader{
			"Content-Type": {contentType},
		},
	}
	// 3. 大小：非空、不超过 50MB
	if err := domainkb.ValidateFile(fh); err != nil {
		return nil, response.Err(http.StatusBadRequest, err.Error())
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = domainkb.DisplayNameFromFilename(filename)
	}
	category := strings.TrimSpace(request.Category)
	validated := &model.ValidatedKnowledgeBaseUpload{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
		Name:        name,
		Category:    category,
	}
	return c.UploadService.Upload(ctx, validated)
}

// getAllKnowledgeBases GET /api/knowledgebase/list：知识库条目列表；当前 501 占位。
func (*KnowledgeBaseController) getAllKnowledgeBases(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getAllKnowledgeBases")
}

// getAllCategories GET /api/knowledgebase/categories：全部分类枚举；当前 501 占位。
func (*KnowledgeBaseController) getAllCategories(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getAllCategories")
}

// getUncategorized GET /api/knowledgebase/uncategorized：未分类条目列表；当前 501 占位。
func (*KnowledgeBaseController) getUncategorized(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getUncategorized")
}

// getStatistics GET /api/knowledgebase/stats：条数/空间等统计；当前 501 占位。
func (*KnowledgeBaseController) getStatistics(_ context.Context) (any, error) {
	return nil, notImplemented("knowledgebase.getStatistics")
}

// getByCategory GET /api/knowledgebase/category/{category}：按分类筛选列表；当前 501 占位。
func (*KnowledgeBaseController) getByCategory(_ context.Context, _ model.KBCategoryPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.getByCategory")
}

// search GET /api/knowledgebase/search：关键词/元数据/向量检索等以主产品为准；当前 501 占位。
func (*KnowledgeBaseController) search(_ context.Context, _ model.KBSearchReq) (any, error) {
	return nil, notImplemented("knowledgebase.search")
}

// queryKnowledgeBaseStream POST /api/knowledgebase/query/stream：对知识库做检索 + LLM 流式回答；当前 501 占位。
func (*KnowledgeBaseController) queryKnowledgeBaseStream(_ context.Context, _ model.KBQueryReq) (any, error) {
	return nil, notImplemented("knowledgebase.queryKnowledgeBaseStream")
}

// queryKnowledgeBase POST /api/knowledgebase/query：对知识库做检索 + 非流式回答，便于联调/自动化；当前 501 占位。
func (*KnowledgeBaseController) queryKnowledgeBase(_ context.Context, _ model.KBQueryReq) (any, error) {
	return nil, notImplemented("knowledgebase.queryKnowledgeBase")
}

// downloadKnowledgeBase GET /api/knowledgebase/{id}/download：下载原始文件；当前 501 占位。
func (*KnowledgeBaseController) downloadKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.downloadKnowledgeBase")
}

// getKnowledgeBase GET /api/knowledgebase/{id}：单条知识库元数据与状态；当前 501 占位。
func (*KnowledgeBaseController) getKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.getKnowledgeBase")
}

// deleteKnowledgeBase DELETE /api/knowledgebase/{id}：删除文件、元数据与向量；当前 501 占位。
func (*KnowledgeBaseController) deleteKnowledgeBase(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.deleteKnowledgeBase")
}

// updateCategory PUT /api/knowledgebase/{id}/category：修改分类；当前 501 占位。
func (*KnowledgeBaseController) updateCategory(_ context.Context, _ model.KBUpdateCategoryReq) (any, error) {
	return nil, notImplemented("knowledgebase.updateCategory")
}

// revectorize POST /api/knowledgebase/{id}/revectorize：重新分块与向量化（如换模型后）；当前 501 占位。
func (*KnowledgeBaseController) revectorize(_ context.Context, _ model.KBIDPathReq) (any, error) {
	return nil, notImplemented("knowledgebase.revectorize")
}

func notImplemented(h string) error {
	return response.Err(http.StatusNotImplemented, errmsg.NotImplemented+": "+h)
}
