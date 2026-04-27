// Package promptprofile 定义简历分析「面试官角色」枚举及与子目录映射（与前端约定同一套 value）。
package promptprofile

import (
	"fmt"
	"strings"
)

// 与前端 interview-guide/src/constants/interviewerRole.ts 中 value 保持一致。
const (
	Backend = "BACKEND"

	// 前端
	Frontend = "FRONTEND"
)

// Parse 校验并规范化表单或 JSON 中的角色字符串。
// 空串、历史「GENERAL」统一视为 FRONTEND，避免旧数据或旧客户端报错。
func Parse(s string) (string, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "GENERAL" {
		return Frontend, nil
	}
	switch s {
	case Backend, Frontend:
		return s, nil
	default:
		return "", fmt.Errorf("invalid interviewer role: %q", s)
	}
}

// PromptSubdir 返回 prompts 下子目录名（小写，与 embed 路径一致）。
func PromptSubdir(role string) string {
	switch strings.TrimSpace(strings.ToUpper(role)) {
	case Backend:
		return "backend"
	case Frontend, "":
		return "frontend"
	default:
		return "frontend"
	}
}
