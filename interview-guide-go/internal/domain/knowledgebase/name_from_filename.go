package knowledgebase

import (
	"path/filepath"
	"strings"
)

// DisplayNameFromFilename 取文件 basename 并去掉扩展名作为知识库展示名；空文件名返回「未命名知识库」。
func DisplayNameFromFilename(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "" || base == "." {
		return "未命名知识库"
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		return base[:i]
	}
	return base
}
