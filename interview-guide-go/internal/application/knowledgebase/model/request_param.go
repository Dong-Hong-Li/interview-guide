package model

// KBListQueryReq GET /api/knowledgebase/list 查询参数：sortBy 排序字段、vectorStatus 状态过滤。
type KBListQueryReq struct {
	SortBy       string `query:"sortBy"`
	VectorStatus string `query:"vectorStatus"`
}

// KBIDPathReq 通过路径 {id} 标识知识库条目。
type KBIDPathReq struct {
	ID int64 `path:"id"`
}

// KBCategoryPathReq 按分类筛选。
// GET /api/knowledgebase/category/{category}
type KBCategoryPathReq struct {
	Category string `path:"category"`
	SortBy   string `query:"sortBy"`
}

// KBUncategorizedQueryReq GET /api/knowledgebase/uncategorized
type KBUncategorizedQueryReq struct {
	SortBy string `query:"sortBy"`
}

// KBSearchReq GET /api/knowledgebase/search?keyword= 搜索知识库的查询参数。
type KBSearchReq struct {
	Keyword string `query:"keyword"`
}

// KBQueryReq POST /api/knowledgebase/query 与 /query/stream 共用 JSON body（与前端 QueryRequest 对齐）。
type KBQueryReq struct {
	KnowledgeBaseIDs []int64 `json:"knowledgeBaseIds" validate:"required"`
	Question         string  `json:"question" validate:"required"`
}

// KBUpdateCategoryReq 更新知识库分类（JSON body + path）。
// PUT /api/knowledgebase/{id}/category
// category 为 null 或省略表示置为未分类（与前端 `category: string | null` 一致）。
type KBUpdateCategoryReq struct {
	ID       int64   `path:"id" json:"-"`
	Category *string `json:"category"`
}

// KBPostUploadRequest 知识库上传请求体，Content-Type: multipart/form-data。
// binding 约定：[]byte + form:"file" 读文件体；Filename/ContentType 由文件头自动填充（勿手填）；name、category 为普通 form 字段。
type KBPostUploadRequest struct {
	Filename    string `json:"filename" form:"-"`
	ContentType string `json:"content_type" form:"-"`
	Content     []byte `json:"content" form:"file" validate:"required"`
	Name        string `json:"name" form:"name"`
	Category    string `json:"category" form:"category"`
}
