// Package controller 知识库 API 路径片段，挂到 /api 下
package controller

const (
	// APIMountPath 知识库域根路径，挂在统一 /api 之下。
	APIMountPath = "/knowledgebase"
)

// 相对已挂载在 /api 上的 chi Router 的 pattern
const (
	// PathPostUpload 上传知识库
	PathPostUpload = "/upload"
	// PathGetList 获取知识库列表
	PathGetList = "/list"
	// PathGetCategories 获取知识库分类
	PathGetCategories = "/categories"
	// PathGetByCategory 按分类筛选
	PathGetByCategory = "/category/{category}"
	// PathGetUncategorized 未分类
	PathGetUncategorized = "/uncategorized"
	// PathGetSearch 搜索
	PathGetSearch = "/search"
	// PathGetStats 统计
	PathGetStats = "/stats"
	// PathPostQueryStream 流式查询
	PathPostQueryStream = "/query/stream"
	// PathPostQuery 非流式查询
	PathPostQuery = "/query"
	// PathGetByIDDownload 下载
	PathGetByIDDownload = "/{id}/download"
	// PathByID 详情 / 删除
	PathByID = "/{id}"
	// PathPutByIDCategory 更新分类
	PathPutByIDCategory = "/{id}/category"
	// PathPostByIDRevectorize 重新向量化
	PathPostByIDRevectorize = "/{id}/revectorize"
)
