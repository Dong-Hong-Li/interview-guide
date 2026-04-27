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

// KBPostUploadNoBody POST /knowledgebase/upload 占位，实现 multipart 时再接真实字段。
type KBPostUploadNoBody struct{}
