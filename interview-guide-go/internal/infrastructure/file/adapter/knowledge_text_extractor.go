package adapter

import (
	"interview-guide-go/internal/application/knowledgebase/repository"
	"interview-guide-go/internal/infrastructure/file"
)

type knowledgeTextExtractor struct{}

// NewKnowledgeTextExtractor 供知识库上传解析正文（PDF/DOCX/TXT/MD）。
func NewKnowledgeTextExtractor() repository.KnowledgeTextExtractor {
	return &knowledgeTextExtractor{}
}

func (*knowledgeTextExtractor) ExtractKnowledgeBaseText(content []byte, filename, contentType string) string {
	return file.ExtractKnowledgeBaseText(content, filename, contentType)
}
