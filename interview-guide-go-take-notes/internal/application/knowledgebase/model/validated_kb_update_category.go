package model

// ValidatedKBUpdateCategory PUT /api/knowledgebase/{id}/category 在校验与 trim 后的入参。
type ValidatedKBUpdateCategory struct {
	ID       int64
	Category string // 空串表示未分类
}
