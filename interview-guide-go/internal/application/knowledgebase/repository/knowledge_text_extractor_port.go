package repository

// KnowledgeTextExtractor 从 PDF/DOCX/TXT/MD 等原件抽取纯文本（与 infrastructure/file 对齐）。
type KnowledgeTextExtractor interface {
	ExtractKnowledgeBaseText(content []byte, filename, contentType string) string
}
