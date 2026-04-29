package domain

const (
	defaultListPageSize = 20
	maxListPageSize     = 100
)

// PageHelper 分页帮助类
type PageHelper struct {
	Page int
	Size int
}

// GetOffset 获取偏移量
func (p *PageHelper) GetOffset() int {
	return (p.Page - 1) * p.Size
}

// normalizeListPaging 规范化分页参数
func NormalizeListPaging(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultListPageSize
	}
	if size > maxListPageSize {
		size = maxListPageSize
	}
	return page, size
}
