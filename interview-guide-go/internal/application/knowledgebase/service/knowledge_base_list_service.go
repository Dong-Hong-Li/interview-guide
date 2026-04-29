package service

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"interview-guide-go/internal/application/knowledgebase/model/results"
	kbrepo "interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
)

// KnowledgeBaseListService 提供知识库的列表 / 分类 / 统计 / 搜索查询。
type KnowledgeBaseListService struct {
	reader kbrepo.KnowledgeBaseReader
}

// NewKnowledgeBaseListService Wire 注入。
func NewKnowledgeBaseListService(r kbrepo.KnowledgeBaseReader) *KnowledgeBaseListService {
	return &KnowledgeBaseListService{reader: r}
}

// GetByID GET /api/knowledgebase/{id}；不存在时返回 (nil, nil) 由 controller 写 404。
func (s *KnowledgeBaseListService) GetByID(ctx context.Context, id int64) (*results.KnowledgeBaseListItem, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	if id < 1 {
		return nil, response.Err(http.StatusBadRequest, "invalid knowledge base id")
	}
	return s.reader.GetKnowledgeBaseByID(ctx, id)
}

// List 根据 vectorStatus 过滤后，按 sortBy 做内存排序（time 已在库侧按上传时间倒序）。
func (s *KnowledgeBaseListService) List(ctx context.Context, vectorStatus *string, sortBy string) ([]results.KnowledgeBaseListItem, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	items, err := s.reader.ListKnowledgeBases(ctx, vectorStatus)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []results.KnowledgeBaseListItem{}
	}
	applySortByKnowledgeBaseList(items, sortBy)
	return items, nil
}

// Categories 返回全部分类名（非空、去重、排序）。
func (s *KnowledgeBaseListService) Categories(ctx context.Context) ([]string, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	cats, err := s.reader.ListDistinctCategories(ctx)
	if err != nil {
		return nil, err
	}
	if cats == nil {
		return []string{}, nil
	}
	return cats, nil
}

// Statistics 返回知识库统计（含 rag_chat_messages 中 USER 角色消息数）。
func (s *KnowledgeBaseListService) Statistics(ctx context.Context) (*results.KnowledgeBaseStats, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	return s.reader.GetStatistics(ctx)
}

// Search 关键字搜索：keyword 为空或仅空白时退回全量列表，便于前端复用同一个端点。
func (s *KnowledgeBaseListService) Search(ctx context.Context, keyword string) ([]results.KnowledgeBaseListItem, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return s.List(ctx, nil, "time")
	}
	items, err := s.reader.SearchKnowledgeBases(ctx, kw)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []results.KnowledgeBaseListItem{}
	}
	return items, nil
}

// ListByCategory GET /api/knowledgebase/category/{category}；category 在 controller 中已校验非空。
func (s *KnowledgeBaseListService) ListByCategory(ctx context.Context, category, sortBy string) ([]results.KnowledgeBaseListItem, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	cat := strings.TrimSpace(category)
	if cat == "" {
		return nil, response.Err(http.StatusBadRequest, errmsg.KnowledgeBaseCategoryEmpty)
	}
	items, err := s.reader.ListByCategory(ctx, cat)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []results.KnowledgeBaseListItem{}
	}
	applySortByKnowledgeBaseList(items, sortBy)
	return items, nil
}

// ListUncategorized GET /api/knowledgebase/uncategorized
func (s *KnowledgeBaseListService) ListUncategorized(ctx context.Context, sortBy string) ([]results.KnowledgeBaseListItem, error) {
	if s == nil || s.reader == nil {
		return nil, response.Err(http.StatusServiceUnavailable, errmsg.KnowledgeBaseListServiceNil)
	}
	items, err := s.reader.ListUncategorized(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []results.KnowledgeBaseListItem{}
	}
	applySortByKnowledgeBaseList(items, sortBy)
	return items, nil
}

// 根据 sortBy 对知识库列表进行排序
func applySortByKnowledgeBaseList(items []results.KnowledgeBaseListItem, sortBy string) {
	sb := strings.ToLower(strings.TrimSpace(sortBy))
	if sb == "" || sb == "time" {
		return
	}
	switch sb {
	case "size":
		sort.Slice(items, func(i, j int) bool { return items[i].FileSize > items[j].FileSize })
	case "access":
		sort.Slice(items, func(i, j int) bool { return items[i].AccessCount > items[j].AccessCount })
	case "question":
		sort.Slice(items, func(i, j int) bool { return items[i].QuestionCount > items[j].QuestionCount })
	default:
		// 未知字段保持库默认顺序
	}
}
