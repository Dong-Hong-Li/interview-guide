package model

// ValidatedKBQuery 已由 Controller 校验后的知识库问答入参（Service 不再做 HTTP 语义校验）。
type ValidatedKBQuery struct {
	KnowledgeBaseIDs []int64
	Question         string
}
