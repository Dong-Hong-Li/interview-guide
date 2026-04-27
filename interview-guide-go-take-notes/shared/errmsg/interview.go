package errmsg

// 面试 API 与主项目 internal/shared/errmsg/interview 对齐的对外提示。
const (
	FindUnfinishedNotFound    = "未找到未完成的面试会话"
	FindUnfinishedBadResumeID = "无效的简历 ID"

	SessionNotFound           = "面试会话不存在"
	GetInterviewSessionFailed = "获取面试会话失败"
	GetCurrentQuestionNoMore  = "面试已结束或已无待答题目"

	CompleteInterviewAlreadyDone = "面试已完成，无法再次交卷"
	CompleteInterviewFailed      = "提前交卷失败"
	// QuestionsNotReady 题目列表仍为空且非明确失败态时的提示（与主项目 SubmitAnswerQuestionsNotReady 语义接近）。
	QuestionsNotReady       = "题目尚未就绪，请稍候重试"
	QuestionsPendingMessage = "题目正在生成中，请稍候"
	QuestionsFailedMessage  = "题目生成失败，请重新开始面试"

	DeleteInterviewSessionBadID    = "无效的会话 ID"
	DeleteInterviewSessionNotFound = "面试会话不存在"
	DeleteInterviewSessionFailed   = "删除面试会话失败"
	DeleteInterviewSessionMessage  = "面试记录已删除"

	SubmitAnswerSessionNotFound   = "面试会话不存在"
	SubmitAnswerSessionClosed     = "面试已结束，无法提交答案"
	SubmitAnswerBadQuestionIndex  = "无效的问题序号"
	SubmitAnswerAnswerEmpty       = "答案不能为空"
	SubmitAnswerQuestionsNotReady = "题目正在生成中，请稍后再试"
	SubmitAnswerPersistFailed     = "保存答案失败"

	GetInterviewReportNotCompleted    = "面试尚未完成，无法获取报告"
	GetInterviewReportEvalFailed      = "面试评估失败"
	GetInterviewReportSyntheticNotice = "综合评估生成中或未接入评估服务；以下为各题答题与得分记录。"

	// InterviewExportPDFFailed 与主项目 internal/shared/errmsg 对齐，供 GET .../export 非字体类失败时返回。
	InterviewExportPDFFailed = "导出面试报告 PDF 失败"

	// InterviewEvalTimeout / InterviewQuestionGenCanceled 供面试评估 Redis 消费者与落库错误文案使用（与主项目 errmsg 对齐）。
	InterviewEvalTimeout         = "评估超时：模型响应超过时限，请稍后重试或重新交卷触发评估"
	InterviewQuestionGenCanceled = "评估已取消"
)
