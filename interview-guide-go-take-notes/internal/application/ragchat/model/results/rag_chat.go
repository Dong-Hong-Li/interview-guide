package results

import (
	"time"

	kbresults "interview-guide-go/internal/application/knowledgebase/model/results"
)

// RagChatSession POST 创建响应，与前端 `RagChatSession` 一致（createdAt 为 RFC3339 字符串化时间）。
type RagChatSession struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	KnowledgeBaseIds []int64   `json:"knowledgeBaseIds"`
	CreatedAt        time.Time `json:"createdAt"`
}

// RagChatSessionListItem GET 列表项，与前端 `RagChatSessionListItem` 一致；updatedAt 为 RFC3339（无 updated_at 时用 created_at）。
type RagChatSessionListItem struct {
	ID                 int64    `json:"id"`
	Title              string   `json:"title"`
	MessageCount       int      `json:"messageCount"`
	KnowledgeBaseNames []string `json:"knowledgeBaseNames"`
	UpdatedAt          string   `json:"updatedAt"`
	IsPinned           bool     `json:"isPinned"`
}

// RagChatMessage 与前端 `RagChatMessage` 一致。
type RagChatMessage struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// RagChatSessionDetail GET 详情，与前端 `RagChatSessionDetail` 一致。
type RagChatSessionDetail struct {
	ID             int64                             `json:"id"`
	Title          string                            `json:"title"`
	KnowledgeBases []kbresults.KnowledgeBaseListItem `json:"knowledgeBases"`
	Messages       []RagChatMessage                  `json:"messages"`
	CreatedAt      time.Time                         `json:"createdAt"`
	UpdatedAt      *time.Time                        `json:"updatedAt,omitempty"`
}
