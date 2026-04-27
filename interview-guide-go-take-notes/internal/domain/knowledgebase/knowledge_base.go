package knowledgebase

import (
	"errors"
	"mime/multipart"
	"strings"
)

// MaxUploadBytes 常量（50MB）
const MaxUploadBytes = 50 * 1024 * 1024

// ValidateFile 与 Java FileValidationService.validateFile（知识库）一致：非空、不超过 50MB。
func ValidateFile(file *multipart.FileHeader) error {
	if file == nil {
		return errors.New("file is nil")
	}
	if file.Size > MaxUploadBytes {
		return errors.New("file size is too large")
	}
	if file.Size == 0 {
		return errors.New("file is empty")
	}
	return nil
}

// ValidateContentType 校验是否为允许的知识库 MIME 类型。
func ValidateContentType(contentType string) error {
	ct := normalizeMIME(contentType)
	if ct == "" {
		return errors.New("content type is empty")
	}
	if !isAllowedContentType(ct) {
		return errors.New("content type is not supported")
	}
	return nil
}

func normalizeMIME(ct string) string {
	ct = strings.TrimSpace(strings.ToLower(ct))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct
}

func isAllowedContentType(ct string) bool {
	switch ct {
	case "application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"text/plain",
		"text/markdown":
		return true
	default:
		return false
	}
}
