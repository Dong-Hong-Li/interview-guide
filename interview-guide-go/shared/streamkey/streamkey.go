package streamkey

// 面试题目异步生成 Stream / 组 / 消息字段。
const (
	// StreamInterviewGenerateQuestions 面试题目异步生成 Stream
	StreamInterviewGenerateQuestions = "interview:generate-questions:stream"
	// GroupInterviewGenerateQuestions 面试题目异步生成 Group
	GroupInterviewGenerateQuestions = "interview:generate-questions:group"

	// StreamFieldIvSessionPublicID 面试题目异步生成 Session Public ID
	StreamFieldIvSessionPublicID = "sessionPublicId"
	// StreamFieldIvResumeID 面试题目异步生成 Resume ID
	StreamFieldIvResumeID = "resumeId"
	// StreamFieldIvQuestionCount 面试题目异步生成 Question Count
	StreamFieldIvQuestionCount = "questionCount"
	// StreamFieldIvResumeText 面试题目异步生成 Resume Text
	StreamFieldIvResumeText = "resumeText"
	// StreamFieldIvHistoricalJSON 面试题目异步生成 Historical JSON
	StreamFieldIvHistoricalJSON = "historicalJson"
	// StreamFieldIvInterviewerRole 面试题目异步生成 Interviewer Role
	StreamFieldIvInterviewerRole = "interviewerRole"

	InterviewGenerateLockKeyPrefix = "interview:generate:lock:"
)

// 面试评估异步任务 Stream / 组 / 字段。
const (
	// StreamInterviewEvaluate 面试评估异步生成 Stream
	StreamInterviewEvaluate = "interview:evaluate:stream"
	// GroupInterviewEvaluate 面试评估异步生成 Group
	GroupInterviewEvaluate = "evaluate-group"
	// StreamFieldEvalSessionID 面试评估异步生成 Session ID
	StreamFieldEvalSessionID = "sessionId"
	// StreamFieldEvalRetryCount 面试评估异步生成 Retry Count
	StreamFieldEvalRetryCount = "retryCount"
)

// 简历分析 Stream / 组 / 字段。
const (
	// StreamResumeAnalyze 简历分析异步生成 Stream
	StreamResumeAnalyze = "resume:analyze:stream"
	// GroupResumeAnalyze 简历分析异步生成 Group
	GroupResumeAnalyze = "analyze-group"
	// StreamFieldResumeID 简历分析异步生成 Resume ID
	StreamFieldResumeID = "resumeId"
	// StreamFieldContent 简历分析异步生成 Content
	StreamFieldContent = "content"
	// StreamFieldRetryCount 简历分析异步生成 Retry Count
	StreamFieldRetryCount = "retryCount"
)

// 知识库向量化异步任务的 Stream 名 / 消费者组 / 字段集中定义。
const (
	// StreamKnowledgeVectorize 知识库分块+向量化任务
	StreamKnowledgeVectorize = "knowledge:vectorize:stream"
	// GroupKnowledgeVectorize 消费者组
	GroupKnowledgeVectorize = "knowledge-vectorize-group"
	// StreamFieldKbID 知识库主键
	StreamFieldKbID = "kbId"
	// StreamFieldKbContent 全文（消费者分块；大文本由 Stream 承载）
	StreamFieldKbContent = "content"
)
