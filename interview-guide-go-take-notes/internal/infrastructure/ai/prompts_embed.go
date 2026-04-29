package ai

import "embed"

// PromptsRoot 嵌入 prompts/ 目录下全部模板文件（简历分析、面试出题、面试评估等）。
// embed 指令必须在包含 prompts 目录的包内执行；adapter 子包通过 fs.ReadFile(PromptsRoot, "prompts/…") 读取。
//
//go:embed prompts
var PromptsRoot embed.FS
