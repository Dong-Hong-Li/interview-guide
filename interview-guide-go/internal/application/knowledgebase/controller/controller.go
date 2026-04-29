package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model"
	"interview-guide-go/internal/application/knowledgebase/service"
	domainkb "interview-guide-go/internal/domain/knowledgebase"
	pdfexport "interview-guide-go/internal/infrastructure/pdf"
	"interview-guide-go/internal/interfaces/http/binding"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

// KnowledgeBaseController 知识库 HTTP 适配层。
type KnowledgeBaseController struct {
	UploadService         *service.UploadKnowledgeBaseService
	ListService           *service.KnowledgeBaseListService
	DeleteService         *service.DeleteKnowledgeBaseService
	DownloadService       *service.DownloadKnowledgeBaseService
	UpdateCategoryService *service.UpdateKnowledgeBaseCategoryService
	RevectorizeService    *service.RevectorizeKnowledgeBaseService
	QueryService          *service.KnowledgeBaseQueryService
}

// Register 将 /api/knowledgebase/* 注册到 r。
func (c *KnowledgeBaseController) Register(r chi.Router) {
	r.Route(APIMountPath, func(sr chi.Router) {
		sr.Post(PathPostUpload, binding.Handle(c.uploadKnowledgeBase))
		sr.Get(PathGetList, binding.Handle(c.getAllKnowledgeBases))
		sr.Get(PathGetCategories, binding.Exec(c.getAllCategories))
		sr.Get(PathGetByCategory, binding.Handle(c.getByCategory))
		sr.Get(PathGetUncategorized, binding.Handle(c.getUncategorized))
		sr.Get(PathGetSearch, binding.Handle(c.search))
		sr.Get(PathGetStats, binding.Exec(c.getStatistics))
		sr.Post(PathPostQueryStream, c.handleQueryKnowledgeBaseStream)
		sr.Post(PathPostQuery, binding.Handle(c.queryKnowledgeBase))
		sr.Get(PathGetByIDDownload, c.handleDownloadKnowledgeBase)
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
		return nil, response.Err(http.StatusBadRequest, "文件超过最大大小50MB")
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

// getAllKnowledgeBases GET /api/knowledgebase/list：与 Java listKnowledgeBases（sortBy、vectorStatus）一致。
func (c *KnowledgeBaseController) getAllKnowledgeBases(ctx context.Context, req model.KBListQueryReq) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	vs := strings.TrimSpace(req.VectorStatus)
	if vs != "" && !domainkb.IsValidVectorStatus(vs) {
		return nil, response.Err(http.StatusBadRequest, "无效的向量化状态: "+vs)
	}
	var vsp *string
	if vs != "" {
		u := strings.ToUpper(vs)
		vsp = &u
	}
	return c.ListService.List(ctx, vsp, strings.TrimSpace(req.SortBy))
}

// getAllCategories GET /api/knowledgebase/categories：全部分类枚举。
func (c *KnowledgeBaseController) getAllCategories(ctx context.Context) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	return c.ListService.Categories(ctx)
}

// getUncategorized GET /api/knowledgebase/uncategorized：未分类条目列表；查询参数 `sortBy` 同 list。
func (c *KnowledgeBaseController) getUncategorized(ctx context.Context, req model.KBUncategorizedQueryReq) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	return c.ListService.ListUncategorized(ctx, strings.TrimSpace(req.SortBy))
}

// getStatistics GET /api/knowledgebase/stats：与 Java getStatistics 一致。
func (c *KnowledgeBaseController) getStatistics(ctx context.Context) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	return c.ListService.Statistics(ctx)
}

// getByCategory GET /api/knowledgebase/category/{category}：按分类筛选；查询参数 `sortBy` 同 list。
func (c *KnowledgeBaseController) getByCategory(ctx context.Context, req model.KBCategoryPathReq) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseCategoryEmpty)
	}
	return c.ListService.ListByCategory(ctx, cat, strings.TrimSpace(req.SortBy))
}

// search GET /api/knowledgebase/search?keyword= 与 Java search、前端 knowledgeBaseApi.search 一致。
func (c *KnowledgeBaseController) search(ctx context.Context, req model.KBSearchReq) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	return c.ListService.Search(ctx, req.Keyword)
}

// queryKnowledgeBase POST /api/knowledgebase/query：对所选知识库做向量检索 + 一次 Chat 作答（JSON Result）。
func (c *KnowledgeBaseController) queryKnowledgeBase(ctx context.Context, req model.KBQueryReq) (any, error) {
	if c == nil || c.QueryService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseQueryServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return nil, err
	}
	v, err := validateKBQueryPayload(&req)
	if err != nil {
		return nil, err
	}
	return c.QueryService.Query(ctx, v)
}

// handleQueryKnowledgeBaseStream POST /api/knowledgebase/query/stream：同上链路但 SSE（text/event-stream），正文为多段 `data:` 与前端 fetch + ReadableStream 解析一致。
func (c *KnowledgeBaseController) handleQueryKnowledgeBaseStream(w http.ResponseWriter, r *http.Request) {
	const maxBody int64 = 4 << 20
	if c == nil || c.QueryService == nil {
		response.ErrJSON(w, http.StatusServiceUnavailable, errmsg.KnowledgeBaseQueryServiceNil)
		return
	}
	if r.Body != nil {
		defer r.Body.Close()
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	var req model.KBQueryReq
	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			response.ErrJSON(w, http.StatusBadRequest, "请求体不能为空")
			return
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.ErrJSON(w, http.StatusRequestEntityTooLarge, "请求体过大")
			return
		}
		response.ErrJSON(w, http.StatusBadRequest, "JSON 格式无效")
		return
	}
	if err := binding.Validate(&req); err != nil {
		response.WriteErr(w, err)
		return
	}
	v, err := validateKBQueryPayload(&req)
	if err != nil {
		response.WriteErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var flushFn func()
	if fl, ok := w.(http.Flusher); ok {
		flushFn = func() { fl.Flush() }
	}

	err = c.QueryService.QueryStream(r.Context(), v, w, flushFn, nil)
	if err != nil {
		response.WriteErr(w, err)
	}
}

// validateKBQueryPayload 去重合法 ID、规整空白后的 KBQueryReq。
func validateKBQueryPayload(req *model.KBQueryReq) (*model.ValidatedKBQuery, error) {
	if req == nil {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseQueryKnowledgeBaseIDsEmpty)
	}
	ids := dedupePositiveKnowledgeBaseIDs(req.KnowledgeBaseIDs)
	if len(ids) == 0 {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseQueryKnowledgeBaseIDsEmpty)
	}
	q := strings.TrimSpace(req.Question)
	if q == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseQueryQuestionEmpty)
	}
	return &model.ValidatedKBQuery{KnowledgeBaseIDs: ids, Question: q}, nil
}

func dedupePositiveKnowledgeBaseIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// handleDownloadKnowledgeBase GET /api/knowledgebase/{id}/download：返回对象存储中的原始文件二进制（非 JSON Result）。
func (c *KnowledgeBaseController) handleDownloadKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	if c == nil || c.DownloadService == nil {
		response.ErrJSON(w, http.StatusServiceUnavailable, errmsg.KnowledgeBaseDownloadServiceNil)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		response.ErrJSON(w, http.StatusBadRequest, "invalid knowledge base id")
		return
	}
	out, err := c.DownloadService.DownloadFile(r.Context(), id)
	if err != nil {
		var he *response.Error
		if errors.As(err, &he) {
			response.ErrJSON(w, he.Code, he.Message)
			return
		}
		response.WriteErr(w, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType)
	w.Header().Set("Content-Disposition", pdfexport.ContentDispositionRFC5987(out.Filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out.Data)
}

// getKnowledgeBase GET /api/knowledgebase/{id}：与 Java getKnowledgeBase 一致，返回 KnowledgeBaseListItem。
func (c *KnowledgeBaseController) getKnowledgeBase(ctx context.Context, req model.KBIDPathReq) (any, error) {
	if c == nil || c.ListService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	item, err := c.ListService.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, response.Err(http.StatusNotFound, errmsg.KnowledgeBaseNotFound)
	}
	return item, nil
}

// deleteKnowledgeBase DELETE /api/knowledgebase/{id}：先尽量删对象存储，再删库行；关联表由 ON DELETE CASCADE 清理。
func (c *KnowledgeBaseController) deleteKnowledgeBase(ctx context.Context, req model.KBIDPathReq) (string, error) {
	if c == nil || c.DeleteService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseDeleteServiceNil)
	}
	if err := c.DeleteService.Delete(ctx, req.ID); err != nil {
		return "", err
	}
	return errmsg.KnowledgeBaseDeleteSuccess, nil
}

// updateCategory PUT /api/knowledgebase/{id}/category：body `category` 为 null/省略时置为未分类。
func (c *KnowledgeBaseController) updateCategory(ctx context.Context, req model.KBUpdateCategoryReq) (string, error) {
	if c == nil || c.UpdateCategoryService == nil {
		return "", response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseUpdateCategoryServiceNil)
	}
	if err := binding.Validate(&req); err != nil {
		return "", err
	}
	if req.ID < 1 {
		return "", response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	cat := ""
	if req.Category != nil {
		cat = strings.TrimSpace(*req.Category)
	}
	if len([]rune(cat)) > 100 {
		return "", response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseCategoryTooLong)
	}
	v := &model.ValidatedKBUpdateCategory{ID: req.ID, Category: cat}
	if err := c.UpdateCategoryService.Update(ctx, v); err != nil {
		return "", err
	}
	return errmsg.KnowledgeBaseUpdateCategorySuccess, nil
}

// revectorize POST /api/knowledgebase/{id}/revectorize：从对象存储取原文、复抽正文，置 vector_status=PENDING 后入队（消费者逻辑与上传后一致）。
func (c *KnowledgeBaseController) revectorize(ctx context.Context, req model.KBIDPathReq) (any, error) {
	if c == nil || c.RevectorizeService == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseRevectorizeServiceNil)
	}
	if req.ID < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	return c.RevectorizeService.Revectorize(ctx, req.ID)
}
