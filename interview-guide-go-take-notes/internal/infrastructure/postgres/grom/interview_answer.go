package model

import (
	"time"

	"gorm.io/gorm"
)

// InterviewAnswer 对应 InterviewAnswerEntity，表 interview_answers。
// SessionID 存父表 interview_sessions 的主键 id（数字），不是对外的 session_id 字符串（与 JPA @JoinColumn 一致）。
// 与 interview_sessions.questions_json 通过 question_index 对齐：每会话每题号至多一行（唯一约束）。
type InterviewAnswer struct {
	// 答案行主键
	ID int64 `gorm:"column:id;primaryKey;autoIncrement"`
	// interview_sessions.id（父表数字主键，非对外 session_id 字符串）
	SessionID int64 `gorm:"column:session_id;not null;index;uniqueIndex:uk_interview_answer_session_question,priority:1"`
	// 题目序号，与 QuestionsJSON 中 QuestionIndex 一致
	QuestionIndex int `gorm:"column:question_index;uniqueIndex:uk_interview_answer_session_question,priority:2;index"`
	// 提交时可冗余存储题干快照（便于审计/导出）
	Question string `gorm:"column:question;type:text"`
	// 题目分类快照
	Category string `gorm:"column:category"`
	// 用户答案正文（暂存或正式提交）
	UserAnswer string `gorm:"column:user_answer;type:text"`
	// 单题得分（评估后写入）
	Score *int `gorm:"column:score"`
	// 单题评语（评估后写入）
	Feedback string `gorm:"column:feedback;type:text"`
	// 单题参考答案要点（可选）
	ReferenceAnswer string `gorm:"column:reference_answer;type:text"`
	// 要点 JSON（可选）
	KeyPointsJSON string `gorm:"column:key_points_json;type:text"`
	// 本条答案首次/最近落库时间
	AnsweredAt time.Time `gorm:"column:answered_at"`
}

func (InterviewAnswer) TableName() string { return "interview_answers" }

func (a *InterviewAnswer) BeforeCreate(_ *gorm.DB) error {
	if a.AnsweredAt.IsZero() {
		a.AnsweredAt = time.Now()
	}
	return nil
}
