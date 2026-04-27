// Package logmsg 集中存放 zap 日志的 message 文案与字段名，避免 cmd 与各层硬编码字符串。
package logmsg

const (
	MsgServerListening           = "HTTP 服务已开始监听"
	MsgListenFatal               = "监听端口失败"
	MsgShutdownWarn              = "收到关闭信号，正在优雅退出"
	MsgServerStopped             = "HTTP 服务已停止"
	MsgLoggerInitFatal           = "logger 初始化失败"
	MsgServerConfigNilSkipWiring = "配置为空，跳过存储/数据库/Redis 等装配"

	MsgStorageStartFailed  = "对象存储服务启动失败"
	MsgPostgresStartFailed = "PostgreSQL 服务启动失败"
	MsgRedisStartFailed    = "Redis 服务启动失败"
	MsgOpenAIStartFailed   = "OpenAI 客户端启动失败"

	MsgResumeAIConsumerEnabled  = "简历 AI 分析后台消费者已启用"
	MsgResumeAIConsumerDisabled = "简历 AI 分析后台消费者启用失败"
	MsgResumeGradeTextTruncated = "简历 AI：简历正文超过上限，已按字符数截断后送模型"

	MsgInterviewEvaluateConsumerEnabled = "面试评估后台消费者已启用"

	MsgKnowledgeVectorizeCreateGroup     = "知识库向量化：创建 Redis Stream 消费者组失败"
	MsgKnowledgeVectorizeConsumerStarted = "知识库向量化消费者已启动"
	MsgKnowledgeVectorizeConsumerStopped = "知识库向量化消费者已停止"
	MsgKnowledgeVectorizeXRead           = "知识库向量化：XREADGROUP 读队列异常"
	MsgKnowledgeVectorizeSkipBad         = "知识库向量化：跳过无效队列消息（缺 kbId 或正文）"
	MsgKnowledgeVectorizeLoadMeta        = "知识库向量化：查询元数据失败"
	MsgKnowledgeVectorizeRowGone         = "知识库向量化：记录已不存在，跳过"
	MsgKnowledgeVectorizeAlreadyDone     = "知识库向量化：已是 COMPLETED，跳过"
	MsgKnowledgeVectorizePersist         = "知识库向量化：回写 COMPLETED 失败"
	MsgKnowledgeVectorizeDone            = "知识库向量化：本条任务处理成功"
	MsgKnowledgeVectorizeConsumerEnabled = "知识库向量化后台消费者已启用"

	MsgInterviewQuestionLLMOK       = "面试出题：LLM 生成题目成功"
	MsgInterviewQuestionLLMDegraded = "面试出题：LLM 调用失败已使用默认题"
	MsgInterviewQuestionLLMBegin    = "面试出题：开始调用模型生成题目"
	// MsgInterviewQuestionLLMFailed 供 GenerateForQueue 等不降级、直接返回 error 的路径；与 Degraded 区分。
	MsgInterviewQuestionLLMFailed = "面试出题：LLM 调用失败"
	MsgInterviewQuestionLLMEmpty  = "面试出题：LLM 返回空题目"

	MsgResumeAnalyzeCreateConsumerGroup = "简历分析：创建 Redis Stream 消费者组失败"
	MsgResumeAnalyzeConsumerStarted     = "简历分析消费者组已启动"
	MsgResumeAnalyzeConsumerStopped     = "简历分析消费者已停止"
	MsgResumeAnalyzeAITaskReceived      = "简历 AI 分析：收到并开始处理队列任务"
	MsgResumeAnalyzeAIBeginGrade        = "简历 AI 分析：开始调用模型评分"
	MsgResumeAnalyzeAIGradeOK           = "简历 AI 分析：模型评分完成"
	MsgResumeAnalyzeXReadGroup          = "简历分析：XREADGROUP 读队列异常"
	MsgResumeAnalyzeSkipBadMessage      = "简历分析：跳过无效队列消息（缺 resumeId 或正文）"
	MsgResumeAnalyzeBadResumeID         = "简历分析：简历 ID 无法解析"
	MsgResumeAnalyzeResumeGone          = "简历分析：数据库中已无该简历记录"
	MsgResumeAnalyzeLoadResume          = "简历分析：加载简历失败"
	MsgResumeAnalyzeMarkProcessing      = "简历分析：更新状态为处理中失败"
	MsgResumeAnalyzeGradeFailed         = "简历分析：AI 打分调用失败"
	MsgResumeAnalyzeMarshalStrengths    = "简历分析：优势项序列化为 JSON 失败"
	MsgResumeAnalyzeInsertAnalysis      = "简历分析：写入 resume_analyses 失败"
	MsgResumeAnalyzeMarkCompleted       = "简历分析：更新状态为已完成失败"
	MsgResumeAnalyzeXAck                = "简历分析：XACK 确认消息失败"
	MsgResumeAnalyzeDone                = "简历 AI 分析：本条任务处理成功"
	MsgResumeDeleteStorageContinue      = "删除对象存储中的简历文件失败，继续删除库内记录"

	MsgInterviewEvaluateCreateConsumerGroup = "面试评估：创建 Redis Stream 消费者组失败"
	MsgInterviewEvaluateConsumerStarted     = "面试评估队列消费者已启动"
	MsgInterviewEvaluateConsumerStopped     = "面试评估队列消费者已停止"
	MsgInterviewEvaluateXReadGroup          = "面试评估：XREADGROUP 读队列异常"
	MsgInterviewEvaluateSkipBadMessage      = "面试评估：跳过无效队列消息"
	MsgInterviewEvaluateSessionGone         = "面试评估：会话已删除，跳过任务"
	MsgInterviewEvaluateNotReady            = "面试评估：会话未交卷或非待评估状态，跳过"
	MsgInterviewEvaluateMarkProcessing      = "面试评估：更新为处理中失败"
	MsgInterviewEvaluateLLMFailed           = "面试评估：LLM 调用或解析失败"
	MsgInterviewEvaluatePersistFailed       = "面试评估：写入报告失败"
	MsgInterviewEvaluateAIBegin             = "面试评估：开始调用模型生成报告"
	MsgInterviewEvaluateSummaryFallback     = "面试评估：二次汇总失败，已使用分批结果聚合"
	MsgInterviewEvaluateDone                = "面试评估：本条任务处理成功"
	MsgInterviewEvaluatePromptsLoad         = "面试评估：加载提示词模板失败，消费者未启动"
)

const (
	FieldAddr = "addr"
	// 日志字段名
	FieldOpenAIBaseURL    = "openaiBaseURL"
	FieldModel            = "model"
	FieldRedis            = "redis"
	FieldPostgres         = "postgres"
	FieldAPIKey           = "apiKey"
	FieldConsumer         = "consumer"
	FieldID               = "id"
	FieldResumeID         = "简历ID"
	FieldOriginalFilename = "简历文件名"
	FieldInterviewerRole  = "面试官角色"
	FieldOverallScore     = "整体评分"
	FieldRuneCount        = "简历字符数"
	FieldMaxRunes         = "最大字符数"
	FieldAIDuration       = "AI 评分耗时"
	// FieldLLMDuration 与主项目 internal/shared/logmsg 的 aiDuration 对齐，供面试题 LLM 等通用模型耗时。
	FieldLLMDuration = "aiDuration"
	// FieldSessionID 对外 session_id；面试评估/会话日志共用。
	FieldSessionID = "sessionId"
	// FieldStatus 通用会话/任务状态值。
	FieldStatus = "status"
)
