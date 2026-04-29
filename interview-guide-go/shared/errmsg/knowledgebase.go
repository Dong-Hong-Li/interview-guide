package errmsg

// 知识库上传与持久化（与 response.Err 搭配）
const (
	KnowledgeBaseUploadServiceNil             = "知识库上传服务未配置"
	KnowledgeBaseListServiceNil               = "知识库列表服务未配置"
	KnowledgeBaseNotFound                     = "知识库不存在"
	KnowledgeBaseDeleteSuccess                = "知识库删除成功"
	KnowledgeBaseCategoryEmpty                = "分类参数不能为空"
	KnowledgeBaseCategoryTooLong              = "分类名称过长"
	KnowledgeBaseUpdateCategoryServiceNil     = "知识库分类更新服务未配置"
	KnowledgeBaseUpdateCategorySuccess        = "知识库分类已更新"
	KnowledgeBaseDeleteServiceNil             = "知识库删除服务未配置"
	KnowledgeBaseDownloadServiceNil           = "知识库下载服务未配置"
	KnowledgeBaseRevectorizeServiceNil        = "知识库重向量化服务未配置"
	KnowledgeBaseNoStorageKey                 = "知识库无对象存储信息，无法下载"
	KnowledgeBaseObjectStorageNotConfigured   = "对象存储未配置"
	KnowledgeBaseWriterNotConfigured          = "知识库持久化未配置"
	KnowledgeBaseTextExtractorNotConfigured   = "知识库正文抽取器未配置"
	KnowledgeBaseExtractTextEmpty             = "无法从文件中提取文本内容，请确保文件格式正确"
	KnowledgeBaseVectorPublisherNotConfigured = "知识库向量化入队未配置，无法完成上传"
	KnowledgeBaseVectorizeChunkEmpty          = "向量化分块后为空"
	KnowledgeBaseChunkAIEmptyChunks           = "AI 分片后无有效正文块（可能全部为异常摘录）"
	KnowledgeBaseEmbeddingCountMismatch       = "向量嵌入返回条数与输入不一致"
	KnowledgeBaseChunkEmbeddingDimMismatch    = "向量嵌入维度与数据库表不一致"

	KnowledgeBaseQueryServiceNil           = "知识库问答服务未配置"
	// KnowledgeBaseQueryDepsNil Service 内部 reader/searcher/writer/chat 等未注入完整。
	KnowledgeBaseQueryDepsNil              = "知识库问答依赖未配置"
	KnowledgeBaseQueryEmbedderNil          = "知识库向量嵌入客户端未配置"
	KnowledgeBaseQueryChatNil              = "知识库问答模型客户端未配置"
	KnowledgeBaseQueryQuestionEmpty        = "问题不能为空"
	KnowledgeBaseQueryKnowledgeBaseIDsEmpty = "至少选择一个知识库"
	KnowledgeBaseVectorNotReadyForQuery    = "所选知识库尚未完成向量化或不可用，请稍后再试"
	// KnowledgeBaseQueryNoHitResponse 检索无有效分块或与阈值不匹配时返回的固定答复，避免 LLM 在无依据时编造内容。
	KnowledgeBaseQueryNoHitResponse = "抱歉，在选定的知识库中未检索到相关信息。请换一个更具体的关键词或补充上下文后再试。"
)

// Embedding 调用失败前缀（写入 vector_error 时会截断总长）。
const KnowledgeBaseEmbeddingFailedPrefix = "向量嵌入失败："

// KnowledgeBaseChunkAIFailedPrefix 知识库 AI 分片（Chat）失败前缀。
const KnowledgeBaseChunkAIFailedPrefix = "知识库 AI 分片失败："

// KnowledgeBasePersistChunksFailedPrefix PG 写入分块向量失败前缀（写入 vector_error 时会截断总长）。
const KnowledgeBasePersistChunksFailedPrefix = "写入分块向量失败："

// KnowledgeBaseVectorizeLoadMetaPrefix 消费者加载 knowledge_bases 元数据失败（写入 vector_error 时会截断总长）。
const KnowledgeBaseVectorizeLoadMetaPrefix = "加载向量化元数据失败："

// KnowledgeBaseVectorizePendingDroppedOnStartup 启动时选择作废 PEL 积压后写入 vector_error（与 Manual 重新入队配合）。
const KnowledgeBaseVectorizePendingDroppedOnStartup = "向量化队列任务因启动策略已作废，请重新上传或重新入队"

// 与 err.Error() 拼接的固定前缀
const (
	FindKnowledgeBaseByHashFailed   = "按文件哈希查询知识库失败："
	UploadKnowledgeBaseFileFailed   = "上传知识库文件失败："
	GetKnowledgeBaseURLFailed       = "获取知识库对象地址失败："
	SaveKnowledgeBaseFailed         = "保存知识库失败："
	SendVectorizeTaskFailed         = "发送向量化任务失败："
	GetKnowledgeBaseObjectFailed    = "读取知识库对象失败："
	DeleteKnowledgeBaseObjectFailed = "删除对象存储文件失败："
	// KnowledgeBaseRevectorizeResetStatusFailed 写入 vector_status=PENDING 失败时与 err.Error() 拼接。
	KnowledgeBaseRevectorizeResetStatusFailed = "重置向量化状态失败："
	// KnowledgeBaseQueryFailedPrefix 调用 Chat Completions 失败时与 err.Error() 拼接。
	KnowledgeBaseQueryFailedPrefix = "知识库问答失败："
)
