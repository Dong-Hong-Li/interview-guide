package model

import (
	"time"

	sharedresume "interview-guide-go/shared/resume"

	"gorm.io/gorm"
)

// Resume 对应 ResumeEntity，表 resumes（简历模块；面试会话 interview_sessions.resume_id 外键引用本表）。
type Resume struct {
	// 简历主键；创建面试时传入的 resumeId
	ID int64 `gorm:"column:id;primaryKey;autoIncrement"`
	// 文件内容哈希，用于去重
	FileHash string `gorm:"column:file_hash;type:varchar(64);uniqueIndex:idx_resume_hash;not null"`
	// 用户上传时的原始文件名
	OriginalFilename string `gorm:"column:original_filename;not null"`
	// 文件大小（字节）
	FileSize *int64 `gorm:"column:file_size"`
	// MIME 类型
	ContentType string `gorm:"column:content_type"`
	// 对象存储键（若有）
	StorageKey string `gorm:"column:storage_key;size:500"`
	// 可访问 URL（若有）
	StorageURL string `gorm:"column:storage_url;size:1000"`
	// 解析后的纯文本简历；面试出题主要依据
	ResumeText string `gorm:"column:resume_text;type:text"`
	// 上传时间
	UploadedAt time.Time `gorm:"column:uploaded_at;autoCreateTime"`
	// 最近访问时间
	LastAccessedAt *time.Time `gorm:"column:last_accessed_at"`
	// 访问次数统计
	AccessCount int `gorm:"column:access_count;default:1"`
	// 简历 AI 分析状态
	AnalyzeStatus string `gorm:"column:analyze_status;size:20;default:PENDING"`
	// 简历分析失败原因
	AnalyzeError string `gorm:"column:analyze_error;size:500"`
	// 与前端下拉 value 一致（如 BACKEND / FRONTEND），决定面试 prompts 人设
	InterviewerRole string `gorm:"column:interviewer_role;size:32;default:FRONTEND"`
}

func (Resume) TableName() string { return "resumes" }

func (r *Resume) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if r.LastAccessedAt == nil {
		r.LastAccessedAt = &now
	}
	if r.AccessCount == 0 {
		r.AccessCount = 1
	}
	if r.AnalyzeStatus == "" {
		r.AnalyzeStatus = string(sharedresume.AnalyzeStatusPending)
	}
	return nil
}
