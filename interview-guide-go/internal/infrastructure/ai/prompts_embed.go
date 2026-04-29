package ai

import "embed"

// PromptsRoot 嵌入 prompts/ 下全部模板（resume|interview|interview-eval × frontend|backend）。
// embed 指令必须在包含 prompts 目录的包内执行；adapter 子包通过 fs.ReadFile(PromptsRoot, "prompts/…") 读取。
//
//go:embed prompts
var PromptsRoot embed.FS
