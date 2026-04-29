// Package logmsg 集中存放 zap 日志的 message 文案与字段名，避免 cmd 与各层硬编码字符串。
package logmsg

const (
	MsgServerListening      = "HTTP 服务已开始监听"
	MsgListenFatal          = "监听端口失败"
	MsgShutdownWarn         = "收到关闭信号，正在优雅退出"
	MsgServerStopped        = "HTTP 服务已停止"
	MsgLoggerInitFatal      = "logger 初始化失败"
	MsgServerConfigNilFatal = "配置为空，进程无法启动"

	MsgStorageStartFailed  = "对象存储服务启动失败"
	MsgPostgresStartFailed = "PostgreSQL 服务启动失败"
	MsgRedisStartFailed    = "Redis 服务启动失败"
	MsgOpenAIStartFailed   = "OpenAI 客户端启动失败"

	// MsgKnowledgeEmbeddingClientFatal KB_EMBEDDING_OPENAI_API_KEY 未配置或 EmbeddingHTTPClient 构造失败时进程退出。
	MsgKnowledgeEmbeddingClientFatal = "知识库 Embedding 客户端初始化失败"

	MsgResumeAIConsumerEnabled  = "简历 AI 分析后台消费者已启用"
	MsgResumeAIConsumerDisabled = "简历 AI 分析后台消费者启用失败"
	MsgResumeGradeTextTruncated = "简历 AI：简历正文超过上限，已按字符数截断后送模型"
	// MsgKnowledgeChunkInputTruncated 知识库 AI 分片前正文超过 KB_CHUNK_MAX_INPUT_RUNES（或沿用简历上限）。
	MsgKnowledgeChunkInputTruncated = "知识库 AI 分片：输入正文超过上限，已按 rune 截断后送模型"
	// MsgKnowledgeChunkAIBegin 与简历 MsgResumeAnalyzeAIBeginGrade 对应：即将 POST chat/completions。
	MsgKnowledgeChunkAIBegin = "知识库 AI 分片：开始调用模型"
	// MsgKnowledgeChunkAIInvokeOK HTTP 成功且 JSON 解析成功后的汇总（chunk 数、token、finish_reason 等）。
	MsgKnowledgeChunkAIInvokeOK = "知识库 AI 分片：模型调用完成"
	// MsgKnowledgeChunkAIInvokeFailed 网络/网关错误或 choices 为空。
	MsgKnowledgeChunkAIInvokeFailed = "知识库 AI 分片：模型调用失败"
	// MsgKnowledgeChunkAIParseFailed 返回体非预期 JSON 或解析失败时；可带 rawPreview。
	MsgKnowledgeChunkAIParseFailed = "知识库 AI 分片：模型返回 JSON 解析失败"

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
	// MsgKnowledgeVectorizeTaskBegin 分块就绪、即将调用 Embed；与 debug 级「链路」配合，info 上只看本条即可识别当前 kbId。
	MsgKnowledgeVectorizeTaskBegin  = "知识库向量化：开始处理当前知识库"
	MsgKnowledgeVectorizeTaskFailed = "知识库向量化：任务失败"
	// MsgKnowledgeVectorizeEnqueued XADD 成功后（含 Redis 返回的 stream 消息 ID）。
	MsgKnowledgeVectorizeEnqueued = "知识库向量化：已写入 Redis Stream"
	// MsgKnowledgeVectorizePulled XREADGROUP 收到一条待处理消息（尚未查库/分块）。
	MsgKnowledgeVectorizePulled = "知识库向量化：已从队列领取消息"
	// MsgKnowledgeVectorizeIdleHint 长时间 Block 超时无新消息：通常表示队列为空，或消息卡在 Pending 未 ACK。
	MsgKnowledgeVectorizeIdleHint = "知识库向量化：轮询中暂无新消息"
	// MsgKnowledgeVectorizePELBacklogHint 启动时 pending>0：多半是上次进程 PID 不同或未 ACK；靠 XAUTOCLAIM(minIdle) 回收或等待超时。
	MsgKnowledgeVectorizePELBacklogHint = "知识库向量化：PEL 有待 ACK 的消息（旧 consumer / 中断）；未读到新任务时请等待回收或排查卡死"
	// MsgKnowledgeVectorizePendingAbortedOnStart 设置 KB_VECTORIZE_ABORT_PENDING_ON_START 后，启动时作废 PEL 并置 DB 为 FAILED。
	MsgKnowledgeVectorizePendingAbortedOnStart = "知识库向量化：已按启动策略作废 PEL 积压（FAILED + XACK）"
	// MsgKnowledgeVectorizeTrace 单条队列任务的分步说明（step / next / outcome，便于排查「卡在哪一步」）。
	MsgKnowledgeVectorizeTrace = "知识库向量化：链路"

	// MsgKnowledgeBaseUploadBegin 新文件已进入存储/落库/入队分支（非 dedup）。
	MsgKnowledgeBaseUploadBegin = "知识库上传：开始处理新文件"
	// MsgKnowledgeBaseUploadDuplicate 与 file_hash 已存在记录重复，未再入队向量化。
	MsgKnowledgeBaseUploadDuplicate = "知识库上传：重复文件，已忽略并仅更新访问次数"
	// MsgKnowledgeBaseUploadOK 落库成功且向量化任务已写入 Redis Stream。
	MsgKnowledgeBaseUploadOK = "知识库上传：成功，已入队向量化"
	// MsgKnowledgeBaseUploadFailed 上传链路任一步失败即将返回错误。
	MsgKnowledgeBaseUploadFailed = "知识库上传：失败"
	// MsgKnowledgeBaseRevectorizeOK 已重置状态并成功写入 Redis Stream；parsedTextRunes 便于核对抽取长度。
	MsgKnowledgeBaseRevectorizeOK = "知识库重向量化：已入队"
	// MsgKnowledgeBaseRevectorizeFailed 抽取为空、置 PENDING、或 XADD 失败（reason：extract_empty | update_pending | enqueue）。
	MsgKnowledgeBaseRevectorizeFailed = "知识库重向量化：失败"
	// MsgKnowledgeBaseQueryBegin 即将对问题做 Embedding 并检索分块（参数含 kbIds、问题长度）。
	MsgKnowledgeBaseQueryBegin = "知识库问答：开始检索"
	// MsgKnowledgeBaseQueryOK 非流式一次生成完成（含 primaryKbId）。
	MsgKnowledgeBaseQueryOK = "知识库问答：完成"
	// MsgKnowledgeBaseQueryFailed Chat Completions 失败或上游网关错误。
	MsgKnowledgeBaseQueryFailed = "知识库问答：失败"

	// MsgRagChatStreamOK RAG 会话流式一轮结束（USER+ASSISTANT 已落库）。
	MsgRagChatStreamOK = "RAG 对话流式：本轮完成"
	// MsgRagChatStreamFailed 向量检索或 Chat 流失败（USER 消息可能已写入）。
	MsgRagChatStreamFailed = "RAG 对话流式：失败"
	// MsgRagChatPersistAssistantFailed 助手消息落库失败（输出可能已送达客户端）。
	MsgRagChatPersistAssistantFailed = "RAG 对话：助手消息落库失败"

	// MsgKnowledgeVectorizeEmbedOK 网关 Embeddings 已全部成功，尚未写 PG（接着 SaveChunks）。
	MsgKnowledgeVectorizeEmbedOK = "知识库向量化：Embedding 网关调用成功"
	// MsgKnowledgeVectorizeChunkAIOutcome AI 分片调用成功并已解析 JSON：须带 chunkCount、exceptionCount、chunkAIDuration。
	MsgKnowledgeVectorizeChunkAIOutcome = "知识库向量化：AI 分片结果汇总"
	// MsgKnowledgeVectorizeChunkAIExceptionItem 与 MsgKnowledgeVectorizeChunkAIOutcome 配套，每条异常一行，便于检索 reason/raw_excerpt。
	MsgKnowledgeVectorizeChunkAIExceptionItem = "知识库向量化：AI 分片异常项"

	// MsgKnowledgeEmbedBatchRun 单次 Embed 调用开始（多批 HTTP 共用一个 runId）。
	MsgKnowledgeEmbedBatchRun = "知识库 Embedding：本次向量请求开始"
	// 「知识库 Embedding」每批 HTTP：发送 → 网关返回（或失败），均经 zap 打到控制台。
	MsgKnowledgeEmbedBatchOutgoing   = "知识库 Embedding：发送批量请求"
	MsgKnowledgeEmbedBatchReturned   = "知识库 Embedding：网关批量响应成功"
	MsgKnowledgeEmbedBatchHTTPFailed = "知识库 Embedding：网关批量响应失败"

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
	// FieldKnowledgeChunkInputRunes 知识库向量化：送 AI 分片的全文 Unicode 字符数（与 Stream body / pg_meta 后正文一致，非简历）。
	FieldKnowledgeChunkInputRunes = "kbChunkInputRunes"
	FieldAIDuration               = "AI 评分耗时"
	// FieldLLMDuration 与主项目 internal/shared/logmsg 的 aiDuration 对齐，供面试题 LLM 等通用模型耗时。
	FieldLLMDuration = "aiDuration"
	// FieldSessionID 对外 session_id；面试评估/会话日志共用。
	FieldSessionID = "sessionId"
	// FieldStatus 通用会话/任务状态值。
	FieldStatus = "status"
	// FieldReason 失败归类：chunk_ai_failed | chunk_empty_after_ai | embedding | embedding_count_mismatch | persist | extract_empty | storage | presign | insert | enqueue 等。
	FieldReason = "reason"
	// FieldVectorizeDuration 单条队列任务从开始处理（分块就绪后）到结束的总耗时。
	FieldVectorizeDuration = "vectorizeDuration"
	// FieldEmbeddingDuration Embed 单次调用耗时（仅向量化链路）。
	FieldEmbeddingDuration = "embeddingDuration"
	// FieldEmbeddingBatchIndex 本轮 Embed 拆分后的批次序号（1-based）。
	FieldEmbeddingBatchIndex = "embeddingBatchIndex"
	// FieldEmbeddingBatchTotal Embed 拆分后的批次总数。
	FieldEmbeddingBatchTotal = "embeddingBatchTotal"
	// FieldEmbeddingSliceStart 当前批次在全文块数组中的起始下标（含）。
	FieldEmbeddingSliceStart = "embeddingSliceStart"
	// FieldEmbeddingSliceEnd 当前批次在全文块数组中的结束下标（含）。
	FieldEmbeddingSliceEnd = "embeddingSliceEnd"
	// FieldEmbeddingBatchRoundTripDuration 单批 POST /embeddings 往返耗时（不含其它批次）。
	FieldEmbeddingBatchRoundTripDuration = "embeddingBatchRoundTrip"
	// FieldEmbeddingDimensionsRequest 请求体传给网关的 dimensions（与 KB_EMBEDDING_DIMENSIONS 一致；v3/v4/3 系列支持）。
	FieldEmbeddingDimensionsRequest = "embeddingDimensions"
	// FieldVectorizeStep 链路阶段：msg_parsed | pg_meta_ok | chunks_split | embed_invoke | embed_ok | pg_save_start | pg_save_ok | redis_xack 等。
	FieldVectorizeStep = "step"
	// FieldVectorizeNext 本步结束后将执行的下一步（人类可读）。
	FieldVectorizeNext = "next"
	// FieldOutcome 本步结果：success | fail（与 HTTP 级 returned 区分）。
	FieldOutcome = "outcome"
	// FieldEmbeddingResponseVectorDim 网关本批返回的每条向量维数（抽样与首条一致时只打一次）。
	FieldEmbeddingResponseVectorDim = "responseVectorDim"
	// FieldEmbeddingResponseCount 网关本批返回的 embedding 条数（应等于 inputCount）。
	FieldEmbeddingResponseCount = "responseEmbeddingCount"
)
