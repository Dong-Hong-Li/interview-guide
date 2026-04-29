package knowledgebase

import (
	"path/filepath"
	"strings"
)

// DisplayNameFromFilename 与 Java KnowledgeBasePersistenceService.extractNameFromFilename 一致：去扩展名；空则「未命名知识库」。
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
