package repository

// 简历解析端口
type TextExtractor interface {
	// 解析简历文本
	ExtractResumeText(content []byte, filename, contentType string) string
}
