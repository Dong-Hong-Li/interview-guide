package knowledgebase

import "strings"

// IsValidVectorStatus 与 VectorStatus 枚举（PENDING/PROCESSING/COMPLETED/FAILED）一致，供 list 查询参数校验。
func IsValidVectorStatus(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "PENDING", "PROCESSING", "COMPLETED", "FAILED":
		return true
	default:
		return false
	}
}
