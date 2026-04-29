package results

// KBQueryResponse POST /api/knowledgebase/query 的成功响应数据体，前端按字段渲染问答结果与命中知识库。
type KBQueryResponse struct {
	Answer              string `json:"answer"`
	KnowledgeBaseID     int64  `json:"knowledgeBaseId"`
	KnowledgeBaseName   string `json:"knowledgeBaseName"`
}
