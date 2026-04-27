package domain_resume

import (
	"errors"
	"interview-guide-go/internal/application/resume/model"
)

// 验证请求参数
func ValidateUploadResumeRequest(request model.UploadResumeRequest) (string, string, []byte, error) {
	filename := request.Filename
	if filename == "" {
		return "", "", nil, errors.New("filename is required")
	}
	contentType := request.ContentType
	if contentType == "" {
		return "", "", nil, errors.New("content type is required")
	}
	content := request.Content
	if len(content) == 0 {
		return "", "", nil, errors.New("content is required")
	}
	return filename, contentType, content, nil
}

// ValidateContentType 校验是否为允许的简历 MIME 类型。
func ValidateContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	switch contentType {
	case "application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return true
	default:
		return false
	}
}

// ValidateContentSize 校验正文不超过 maxBytes；maxBytes <= 0 表示不限制。
func ValidateContentSize(content []byte, maxBytes int64) bool {
	if maxBytes <= 0 {
		return true
	}
	return int64(len(content)) <= maxBytes
}
