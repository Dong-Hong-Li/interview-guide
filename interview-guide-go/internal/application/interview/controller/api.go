// Package controller 定义面试模拟（会话）API 的路径片段；与 Register 中的 r.Route 组合后，
// 完整前缀一般为 /api + APIMountPath（与简历模块 shared/resume.APIMountPath 约定一致）。
package controller

const (
	// APIMountPath 面试域根路径，挂在统一 /api 之下。
	APIMountPath = "/interview"
)

const (
	// InterviewNewSessions 针对某份简历开启一轮模拟面试（创建会话）。
	InterviewNewSessions = "/sessions"
	// PathGetUnfinishedByResume 按简历 ID 拉取尚未结束的会话（继续答到一半的那场）。
	PathGetUnfinishedByResume = "/sessions/unfinished/{resumeId}"
	// PathGetSessionQuestion 取当前会话下一题/当前题（供答题页展示）。
	PathGetSessionQuestion = "/sessions/{sessionId}/question"
	// PathGetSessionReport 会话结束后的评分与报告摘要。
	PathGetSessionReport = "/sessions/{sessionId}/report"
	// PathGetSessionDetails 会话元数据与题目、答案时间线等详情。
	PathGetSessionDetails = "/sessions/{sessionId}/details"
	// PathGetSessionExport 导出本场会话记录（如 PDF/Markdown，由实现决定）。
	PathGetSessionExport = "/sessions/{sessionId}/export"
	// PathDeleteSession 删除一场会话及其侧存数据。
	PathDeleteSession = "/sessions/{sessionId}"
	// PathSessionAnswers POST 提交单题答案（进入下一题）。
	PathSessionAnswers = "/sessions/{sessionId}/answers"
	// PathPostSessionComplete 主动结束会话（全部答完或用户结束）。
	PathPostSessionComplete = "/sessions/{sessionId}/complete"
	// PathGetSession 按 sessionId 取会话基本信息。
	PathGetSession = "/sessions/{sessionId}"
)
