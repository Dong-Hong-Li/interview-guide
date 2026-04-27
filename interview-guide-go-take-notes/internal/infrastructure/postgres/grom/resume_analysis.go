package model

import (
	"time"

	"gorm.io/gorm"
)

// ResumeAnalysis 对应 ResumeAnalysisEntity，表 resume_analyses。
type ResumeAnalysis struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ResumeID        int64     `gorm:"column:resume_id;not null;index"`
	OverallScore    *int      `gorm:"column:overall_score"`
	ContentScore    *int      `gorm:"column:content_score"`
	StructureScore  *int      `gorm:"column:structure_score"`
	SkillMatchScore *int      `gorm:"column:skill_match_score"`
	ExpressionScore *int      `gorm:"column:expression_score"`
	ProjectScore    *int      `gorm:"column:project_score"` //
	Summary         string    `gorm:"column:summary;type:text"`
	StrengthsJSON   string    `gorm:"column:strengths_json;type:text"`
	SuggestionsJSON string    `gorm:"column:suggestions_json;type:text"`
	AnalyzedAt      time.Time `gorm:"column:analyzed_at"`
}

func (ResumeAnalysis) TableName() string { return "resume_analyses" }

// BeforeCreate 在创建前设置 AnalyzedAt 为当前时间
func (a *ResumeAnalysis) BeforeCreate(_ *gorm.DB) error {
	if a.AnalyzedAt.IsZero() {
		a.AnalyzedAt = time.Now()
	}
	return nil
}
