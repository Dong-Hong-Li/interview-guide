package results

import "time"

// KnowledgeBaseListItem 与 example KnowledgeBaseListItemDTO 及前端列表字段一致。
type KnowledgeBaseListItem struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Category         *string    `json:"category,omitempty"`
	OriginalFilename string     `json:"originalFilename"`
	FileSize         int64      `json:"fileSize"`
	ContentType      string     `json:"contentType"`
	UploadedAt       time.Time  `json:"uploadedAt"`
	LastAccessedAt   *time.Time `json:"lastAccessedAt,omitempty"`
	AccessCount      int        `json:"accessCount"`
	QuestionCount    int        `json:"questionCount"`
	VectorStatus     string     `json:"vectorStatus"`
	VectorError      string     `json:"vectorError,omitempty"`
	ChunkCount       int        `json:"chunkCount"`
}

// KnowledgeBaseStats 与 example KnowledgeBaseStatsDTO 一致。
type KnowledgeBaseStats struct {
	TotalCount         int64 `json:"totalCount"`
	TotalQuestionCount int64 `json:"totalQuestionCount"`
	TotalAccessCount   int64 `json:"totalAccessCount"`
	CompletedCount     int64 `json:"completedCount"`
	ProcessingCount    int64 `json:"processingCount"`
}
