package result

// ResumeStatsResult GET /api/resumes/statistics 响应体，与前端 `history.ResumeStats` 一致。
type ResumeStatsResult struct {
	TotalCount          int64 `json:"totalCount"`
	TotalInterviewCount int64 `json:"totalInterviewCount"`
	TotalAccessCount    int64 `json:"totalAccessCount"`
}
