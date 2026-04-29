package model

import (
	"time"

	"gorm.io/gorm"
)

// 模拟面试持久化（schema/04_interview.sql）共两张表：
//  1. interview_sessions — 一次面试会话：对外 session_id、关联简历、题目 JSON、答题游标、会话状态、评估汇总等。
//  2. interview_answers — 该会话下每道题的作答与单题评分（session_id 存本表主键 id，非对外 session_id 字符串）。
//
// 另通过 resume_id 引用 resumes（简历模块表）。
//
// InterviewSession 对应 InterviewSessionEntity，表 interview_sessions。
type InterviewSession struct {
	// 表主键（内部 ID；对外接口使用 SessionID）
	ID int64 `gorm:"column:id;primaryKey;autoIncrement"`
	// 对外会话标识（URL/API 中的 sessionId）
	SessionID string `gorm:"column:session_id;size:36;uniqueIndex;not null"`
	// 关联简历主键 resumes.id
	ResumeID int64 `gorm:"column:resume_id;not null;index"`
	// 约定总题数（可与 questions_json 长度一致；异步出题时可用于展示）
	TotalQuestions *int `gorm:"column:total_questions"`
	// 当前应答题在题目列表中的下标（0-based）
	CurrentQuestionIndex int `gorm:"column:current_question_index;default:0"`
	// 会话生命周期：CREATED / QUESTIONS_PENDING / QUESTIONS_FAILED / IN_PROGRESS / COMPLETED / EVALUATED
	Status string `gorm:"column:status;size:20;default:CREATED;index"`
	// 全部面试题 JSON 数组（题干、类型等；用户答案在 interview_answers）
	QuestionsJSON string `gorm:"column:questions_json;type:text"`
	// 整场面试综合得分（评估完成后写入）
	OverallScore *int `gorm:"column:overall_score"`
	// 整场综合评语
	OverallFeedback string `gorm:"column:overall_feedback;type:text"`
	// 优势列表 JSON（评估结果）
	StrengthsJSON string `gorm:"column:strengths_json;type:text"`
	// 改进建议 JSON（评估结果）
	ImprovementsJSON string `gorm:"column:improvements_json;type:text"`
	// 参考要点等 JSON（评估/导出用）
	ReferenceAnswersJSON string `gorm:"column:reference_answers_json;type:text"`
	// 会话创建时间
	CreatedAt time.Time `gorm:"column:created_at;index"`
	// 用户交卷完成时间（可为空）
	CompletedAt *time.Time `gorm:"column:completed_at"`
	// 评估流水线状态：PENDING / PROCESSING / COMPLETED / FAILED（见 const InterviewEvaluateStatus*）
	EvaluateStatus string `gorm:"column:evaluate_status;size:20"`
	// 错误说明：评估失败或队列出题失败等场景共用本列（最多约 500 字符）
	EvaluateError string `gorm:"column:evaluate_error;size:500"`
}

func (InterviewSession) TableName() string { return "interview_sessions" }

func (s *InterviewSession) BeforeCreate(_ *gorm.DB) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	return nil
}
