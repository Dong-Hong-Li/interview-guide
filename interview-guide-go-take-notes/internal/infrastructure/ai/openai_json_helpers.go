package ai

import (
	"strings"

	"github.com/openai/openai-go/shared"
)

// PtrJSONObjectFormat 返回 Chat Completions 的「响应必须为 JSON 对象」格式参数。
//
// 业务场景：知识库分片、简历结构化打分等需要模型输出可 json.Unmarshal 的负荷；
// 早期模型不支持 json_schema / Structured Outputs 时，仍可通过 response_format=json_object 约束大致形态，
// 再由 ExtractJSONObject 容忍模型在 JSON 前后输出的 Markdown 或说明文字。
func PtrJSONObjectFormat() *shared.ResponseFormatJSONObjectParam {
	rf := shared.NewResponseFormatJSONObjectParam()
	return &rf
}

// ExtractJSONObject 从模型原始字符串中提取第一段 `{...}` 子串。
//
// 背景：即使用了 JSON 模式，部分网关仍会把内容包在 ```json 代码块里，或在 JSON 前写一句「以下是结果：」；
// 上层解析前先做截取，避免.Unmarshal 整个失败。若字符串本身以 `{` 开头则保持不变。
func ExtractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i > 0 {
		s = s[i:]
	}
	if j := strings.LastIndex(s, "}"); j >= 0 && j < len(s)-1 {
		s = s[:j+1]
	}
	return s
}
