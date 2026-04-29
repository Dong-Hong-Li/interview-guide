package repository

import (
	"context"
	"errors"
	"time"
)

// ErrResumeNotFound 队列消费者按 ID 加载简历时未命中。
var ErrResumeNotFound = errors.New("resume not found")

// ErrResumeTextUnavailable 重分析时库内与对象存储均无法得到可用简历正文，调用方据此返回明确错误。
var ErrResumeTextUnavailable = errors.New("无法获取简历文本内容")

// ResumeInsert 写入简历表所需字段（与 ORM 表解耦，由基础设施映射为行）。
type ResumeInsert struct {
	// 文件内容哈希，用于去重
	FileHash string
	// 用户上传时的原始文件名
	OriginalFilename string
	// 文件大小（字节）
	FileSize int64
	// 文件类型
	ContentType string
	// 对象存储键（若有）
	StorageKey string
	// 可访问 URL（若有）
	StorageURL string
	// 解析后的纯文本简历；面试出题主要依据
	ResumeText string
	// 与前端下拉 value 一致（如 BACKEND / FRONTEND），决定面试 prompts 人设
	InterviewerRole string
	// 简历分析状态
	AnalyzeStatus string
	// 简历分析失败原因
	AnalyzeError string
	// 简历分析时间
	AnalyzeTime time.Time
}

// ResumeListRow 列表查询返回行（与 ORM 解耦，由基础设施填充）。
type ResumeListRow struct {
	// 简历主键
	ID int64
	// 用户上传时的原始文件名
	OriginalFilename string
	// 上传时间
	UploadedAt time.Time
	// 文件大小（字节）
	FileSize int64
	// 可访问 URL（若有）
	StorageURL string
	// 简历分析状态
	AnalyzeStatus string
	// 简历分析失败原因
	AnalyzeError string
	// 与前端下拉 value 一致（如 BACKEND / FRONTEND），决定面试 prompts 人设
	InterviewerRole string
	// 最近访问时间
	LastAccessedAt *time.Time
	// 访问次数
	AccessCount int
}

// ExistingResume 去重查询的返回视图（仅包含上传用例需要复用的字段）。
type ExistingResume struct {
	// 简历主键
	ID int64
	// 用户上传时的原始文件名
	OriginalFilename string
	// 对象存储键（若有）
	StorageKey string
	// 可访问 URL（若有）
	StorageURL string
	// 与前端下拉 value 一致（如 BACKEND / FRONTEND），决定面试 prompts 人设
	InterviewerRole string
}

// ResumeForDetail 详情页简历头信息（与 GET /api/resumes/{id}/detail 元数据一致，不含子表）。
type ResumeForDetail struct {
	ID               int64
	OriginalFilename string
	FileSize         int64
	ContentType      string
	// StorageKey 对象存储键；resume_text 为空时重分析需用其下载与重新解析
	StorageKey      string
	StorageURL      string
	ResumeText      string
	InterviewerRole string
	AnalyzeStatus   string
	AnalyzeError    string
	UploadedAt      time.Time
	AccessCount     int
}

// ResumeAnalysisListRow 一条简历分析历史（与 resume_analyses 行对应，供结果 JSON 转换）。
type ResumeAnalysisListRow struct {
	ID              int64
	OverallScore    *int
	ContentScore    *int
	StructureScore  *int
	SkillMatchScore *int
	ExpressionScore *int
	ProjectScore    *int
	Summary         string
	StrengthsJSON   string
	SuggestionsJSON string
	AnalyzedAt      time.Time
}

// ResumeAnalysisInput 写入 resume_analyses 一行的应用层入参（与 ORM 解耦）。
type ResumeAnalysisInput struct {
	// 简历主键
	ResumeID int64
	// 整体评分
	OverallScore *int
	// 内容评分
	ContentScore *int
	// 结构评分
	StructureScore *int
	// 技能匹配评分
	SkillMatchScore *int
	// 表达评分
	ExpressionScore *int
	// 项目评分
	ProjectScore *int
	// 总结
	Summary string
	// 优势 JSON
	StrengthsJSON string
	// 建议 JSON
	SuggestionsJSON string
}

// ResumeWriter 简历持久化写入端口（由 postgres/mapper 等实现）。
type ResumeWriter interface {
	// InsertResume 插入一行并返回自增主键 id。
	InsertResume(ctx context.Context, in *ResumeInsert) (id int64, err error)

	// FindByFileHash 根据文件哈希查找已存在简历；未找到返回 (nil, nil)。
	FindByFileHash(ctx context.Context, fileHash string) (*ExistingResume, error)

	// GetResumeForAnalyze 按主键读简历，供分析消费者取 InterviewerRole 等；未找到返回 ErrResumeNotFound。
	GetResumeForAnalyze(ctx context.Context, resumeID int64) (*ExistingResume, error)

	// GetResumeForDetail 读详情页头字段（全列）；未找到返回 ErrResumeNotFound。
	GetResumeForDetail(ctx context.Context, resumeID int64) (*ResumeForDetail, error)

	// ListAnalysesByResumeID 该简历下全部分析记录，按 analyzed_at 降序。
	ListAnalysesByResumeID(ctx context.Context, resumeID int64) ([]ResumeAnalysisListRow, error)

	// UpdateAnalyzeStatus 更新简历分析状态（COMPLETE / FAILED）。
	UpdateAnalyzeStatus(ctx context.Context, resumeID int64, status string, errorMessage string) error

	// UpdateResumeText 只更新 resume_text 列，用于重分析时回灌从对象存储重新解析后的文本，避免再次拉取。
	UpdateResumeText(ctx context.Context, resumeID int64, resumeText string) error

	// InsertResumeAnalysis 写入一条 AI 分析结果（resume_analyses）。
	InsertResumeAnalysis(ctx context.Context, in *ResumeAnalysisInput) error

	// InsertAnalyzeJob 插入简历分析任务
	InsertAnalyzeJob(ctx context.Context, job *AnalyzeJob) error

	// ListResumes 分页查询简历；page 从 1 开始，size 为每页条数；返回当前页数据与全表总数（用于 total）。
	ListResumes(ctx context.Context, page, size int) (rows []ResumeListRow, total int64, err error)

	// AggregateResumeGlobalStats 全库简历条数、面试会话总数、简历 access_count 之和（PostgreSQL）。
	AggregateResumeGlobalStats(ctx context.Context) (totalCount, totalInterviewCount, totalAccessCount int64, err error)

	// DeleteResumeByID 删除简历
	DeleteResumeByID(ctx context.Context, id int64) error

	// DeleteResumeAnalysisByResumeID 删除简历分析结果
	DeleteResumeAnalysisByResumeID(ctx context.Context, resumeID int64) error
}
