package repository

import (
	"context"

	"interview-guide-go/internal/application/knowledgebase/model/results"
)

// KnowledgeBaseFileRef 下载/删除时使用的对象键与元数据（不进入列表 JSON）。
type KnowledgeBaseFileRef struct {
	StorageKey       string
	OriginalFilename string
	ContentType      string
}

// KnowledgeBaseReader 知识库只读查询（列表、分类、统计），由 postgres mapper 实现。
type KnowledgeBaseReader interface {
	// GetKnowledgeBaseFileRef 按 id 取存储 key 与展示文件名；不存在返回 (nil, nil)。
	GetKnowledgeBaseFileRef(ctx context.Context, id int64) (*KnowledgeBaseFileRef, error)
	// GetKnowledgeBaseByID 单条详情；不存在返回 (nil, nil)。
	GetKnowledgeBaseByID(ctx context.Context, id int64) (*results.KnowledgeBaseListItem, error)
	// ListKnowledgeBases 与 Java listKnowledgeBases 一致：按 uploaded_at 倒序；若 vectorStatus 非 nil 则按状态过滤（大写比较）。
	ListKnowledgeBases(ctx context.Context, vectorStatus *string) ([]results.KnowledgeBaseListItem, error)
	// ListDistinctCategories 与 Java findAllCategories 一致：非空分类去重、字典序。
	ListDistinctCategories(ctx context.Context) ([]string, error)
	// GetStatistics 与 Java getStatistics 一致：条数、USER 消息数、访问合计、向量化态计数。
	GetStatistics(ctx context.Context) (*results.KnowledgeBaseStats, error)
	// SearchKnowledgeBases 与 Java searchByKeyword 一致：name / original_filename 子串匹配（PostgreSQL ILIKE），uploaded_at 倒序。
	SearchKnowledgeBases(ctx context.Context, keyword string) ([]results.KnowledgeBaseListItem, error)
	// ListByCategory 与分类名精确匹配（TRIM 后与库内展示一致），uploaded_at 倒序。
	ListByCategory(ctx context.Context, category string) ([]results.KnowledgeBaseListItem, error)
	// ListUncategorized 仅 category 为空或仅空白（与 findAllCategories「空串视为无分类」一致），uploaded_at 倒序。
	ListUncategorized(ctx context.Context) ([]results.KnowledgeBaseListItem, error)
}
