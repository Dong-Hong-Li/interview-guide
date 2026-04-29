package results

// KBQueryResponse POST /api/knowledgebase/query 成功体（与前端 QueryResponse、Java QueryResponse 对齐）。
type KBQueryResponse struct {
	Answer              string `json:"answer"`
	KnowledgeBaseID     int64  `json:"knowledgeBaseId"`
	KnowledgeBaseName   string `json:"knowledgeBaseName"`
}
