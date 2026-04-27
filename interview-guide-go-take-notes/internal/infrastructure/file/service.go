package file

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tsawler/tabula"
)

// watermarkRe 匹配简历平台嵌入 PDF 的追踪水印（长 hex+base64 混合串，常以 ~ 结尾）。
var watermarkRe = regexp.MustCompile(`(?m)^\s*[0-9a-fA-F]{16,}[A-Za-z0-9+/=~]{8,}\s*$`)

// 从 PDF / DOCX 提取纯文本
//
// PDF 优先使用 pdftotext（poppler-utils），质量远高于纯 Go 解析；不可用时回退 tabula。
// DOCX 使用 tabula。
func ExtractResumeText(content []byte, filename, contentType string) string {
	if len(content) == 0 {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(filename))
	ct := strings.ToLower(strings.TrimSpace(contentType))

	var raw string
	switch {
	case ext == ".pdf" || strings.Contains(ct, "application/pdf"):
		if text := extractPDFViaPoppler(content); text != "" {
			raw = text
		} else {
			raw = extractViaTabula(content, ".pdf")
		}
	case ext == ".docx" ||
		ct == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		raw = extractViaTabula(content, ".docx")
	default:
		return ""
	}
	return stripWatermarks(raw)
}

// stripWatermarks 去除简历平台嵌入的追踪水印行，并合并多余空行。
func stripWatermarks(s string) string {
	cleaned := watermarkRe.ReplaceAllString(s, "")
	// 合并连续空行为单个空行
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(cleaned)
}

// extractPDFViaPoppler 使用 pdftotext（poppler-utils）提取 PDF 文本。
// -layout 保留原始版面顺序，输出质量接近 PDF24 网页版。
func extractPDFViaPoppler(content []byte) string {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return ""
	}
	tmp, err := os.CreateTemp("", "resume-*.pdf")
	if err != nil {
		return ""
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return ""
	}
	tmp.Close()

	var stdout bytes.Buffer
	cmd := exec.Command("pdftotext", "-layout", tmpPath, "-")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

// extractViaTabula 将内存字节写入临时文件，用 tabula 提取文本后清理。
func extractViaTabula(content []byte, ext string) string {
	if ext == "" {
		ext = ".pdf"
	}
	tmp, err := os.CreateTemp("", "resume-*"+ext)
	if err != nil {
		return ""
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return ""
	}
	tmp.Close()

	text, _, err := tabula.Open(tmpPath).
		ExcludeHeadersAndFooters().
		JoinParagraphs().
		Text()
	if err != nil {
		return ""
	}
	return text
}

// ExtractKnowledgeBaseText 从知识库 supported 原件抽取纯文本：TXT/MD 直接按 UTF-8；PDF/DOCX 等同简历路径。
func ExtractKnowledgeBaseText(content []byte, filename, contentType string) string {
	if len(content) == 0 {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(filename))
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case ext == ".txt" || ext == ".md":
		return strings.TrimSpace(string(content))
	case strings.HasPrefix(ct, "text/plain"), strings.HasPrefix(ct, "text/markdown"):
		return strings.TrimSpace(string(content))
	default:
		return ExtractResumeText(content, filename, contentType)
	}
}
