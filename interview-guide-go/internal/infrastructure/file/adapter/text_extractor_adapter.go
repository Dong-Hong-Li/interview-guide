package adapter

import (
	"interview-guide-go/internal/application/resume/repository"
	"interview-guide-go/internal/infrastructure/file"
)

// TextExtractorAdapter 文本提取适配器
type TextExtractorAdapter struct {
}

// NewTextExtractorAdapter 创建文本提取适配器
func NewTextExtractorAdapter() repository.TextExtractor {
	return &TextExtractorAdapter{}
}

func (s *TextExtractorAdapter) ExtractResumeText(content []byte, filename, contentType string) string {
	return file.ExtractResumeText(content, filename, contentType)
}
