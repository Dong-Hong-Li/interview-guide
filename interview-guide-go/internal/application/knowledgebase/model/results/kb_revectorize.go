package results

// RevectorizeKnowledgeBaseResponse POST /api/knowledgebase/{id}/revectorize 成功体（与前端约定最小字段）。
type RevectorizeKnowledgeBaseResponse struct {
	ID           int64  `json:"id"`
	VectorStatus string `json:"vectorStatus"`
	// ParsedTextRunes 抽取后送入队列的正文 Unicode 字符数（与 Upload 入队一致，不落库全文）。
	ParsedTextRunes int `json:"parsedTextRunes"`
}
