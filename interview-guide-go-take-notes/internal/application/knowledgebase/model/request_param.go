package model

// KBIDPathReq 通过路径 {id} 标识知识库条目。
type KBIDPathReq struct {
	ID int64 `path:"id"`
}

// KBCategoryPathReq 按分类筛选。
// GET /api/knowledgebase/category/{category}
type KBCategoryPathReq struct {
	Category string `path:"category"`
}

// KBSearchReq 搜索知识库。
// GET /api/knowledgebase/search?q=...
type KBSearchReq struct {
	Q string `query:"q"`
}

// KBQueryReq 知识库 RAG 查询（JSON body）。占位 501 阶段无字段。
type KBQueryReq struct{}

// KBUpdateCategoryReq 更新知识库分类（JSON body + path）。
// PUT /api/knowledgebase/{id}/category
type KBUpdateCategoryReq struct {
	ID       int64  `path:"id" json:"-"`
	Category string `json:"category"`
}

// KBPostUploadRequest POST /api/knowledgebase/upload，Content-Type: multipart/form-data。
// 与 Java KnowledgeBaseController.uploadKnowledgeBase（file、name、category）及前端 knowledgeBaseApi.uploadKnowledgeBase 一致。
// binding：[]byte + form:"file" 读文件体；Filename/ContentType 由文件头自动填充（勿手填）；name、category 为普通 form 字段。
type KBPostUploadRequest struct {
	Filename    string `json:"filename" form:"-"`
	ContentType string `json:"content_type" form:"-"`
	Content     []byte `json:"content" form:"file" validate:"required"`
	Name        string `json:"name" form:"name"`
	Category    string `json:"category" form:"category"`
}
