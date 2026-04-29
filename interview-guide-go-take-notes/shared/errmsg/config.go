package errmsg

// 环境变量校验失败时的错误文案（可与 errors.New 搭配，供日志与排障）。
const (
	ConfigServerHostRequired       = "未设置 SERVER_HOST（监听地址）"
	ConfigServerPortInvalid        = "SERVER_PORT 无效，须为正整数"
	ConfigServerReadTimeoutInvalid = "SERVER_READ_TIMEOUT_SECONDS 无效，须为正整数（秒）"

	ConfigDatabaseURLRequired      = "未设置 DATABASE_URL"
	ConfigPostgresHostRequired     = "未设置 POSTGRES_HOST"
	ConfigPostgresPortRequired     = "未设置 POSTGRES_PORT"
	ConfigPostgresUserRequired     = "未设置 POSTGRES_USER"
	ConfigPostgresPasswordRequired = "未设置 POSTGRES_PASSWORD"
	ConfigPostgresDBNameRequired   = "未设置 POSTGRES_DB"
	ConfigPostgresSSLModeRequired  = "未设置 POSTGRES_SSLMODE"

	ConfigRedisHostRequired = "未设置 REDIS_HOST"
	ConfigRedisPortRequired = "未设置 REDIS_PORT"
	ConfigRedisDBInvalid    = "REDIS_DB 无效，须为不小于 0 的整数"

	ConfigStorageEndpointRequired      = "未设置 APP_STORAGE_ENDPOINT"
	ConfigStorageAccessKeyRequired     = "未设置 APP_STORAGE_ACCESS_KEY"
	ConfigStorageSecretKeyRequired     = "未设置 APP_STORAGE_SECRET_KEY"
	ConfigStorageBucketRequired        = "未设置 APP_STORAGE_BUCKET"
	ConfigStorageRegionRequired        = "未设置 APP_STORAGE_REGION"
	ConfigStoragePresignExpiresInvalid = "APP_STORAGE_PRESIGN_GET_EXPIRES_SEC 无效，须为正整数（秒）"

	ConfigOpenAIAPIKeyRequired                = "未设置 OPENAI_API_KEY"
	ConfigOpenAIBaseURLRequired               = "未设置 OPENAI_BASE_URL"
	ConfigMoonshotAPIKeyRequired              = "未设置 MOONSHOT_API_KEY"
	ConfigAIModelRequired                     = "未设置 AI_MODEL"
	ConfigResumeAIMaxRunesRequired            = "未设置 RESUME_AI_MAX_RUNES（须为正整数）"
	ConfigResumeAIMaxRunesInvalid             = "RESUME_AI_MAX_RUNES 无效，须为正整数"
	ConfigResumeAIMaxCompletionTokensRequired = "未设置 RESUME_AI_MAX_COMPLETION_TOKENS（须为正整数）"
	ConfigResumeAIMaxTokensInvalid            = "RESUME_AI_MAX_COMPLETION_TOKENS 无效，须为正整数"
	ConfigResumeAITemperatureInvalid          = "RESUME_AI_TEMPERATURE 无效，须为不小于 0 的数"

	ConfigCORSAllowedOriginsRequired = "未设置 CORS_ALLOWED_ORIGINS（逗号分隔；开发可设为 http://localhost:5173）"

	ConfigStartupSnapshotMarshalFail = "启动时序列化配置快照失败"
)

// 进程启动阶段 log.Fatalf 前缀（与具体校验错误拼接）。
const (
	LogFatalValidateServerConfig   = "[配置] 校验 server 失败"
	LogFatalValidateDatabaseConfig = "[配置] 校验 database 失败"
	LogFatalValidateRedisConfig    = "[配置] 校验 redis 失败"
	LogFatalValidateStorageConfig  = "[配置] 校验 storage 失败"
	LogFatalValidateOpenAIConfig   = "[配置] 校验 openai 失败"
	LogFatalValidateCORSConfig     = "[配置] 校验 CORS 失败"
)

// 配置为空时的错误文案。
const (
	ConfigOpenAIServiceStartFailed = "大模型客户端启动失败"

	ConfigKnowledgeEmbeddingGatewayBaseURLRequired = "未设置 KB_EMBEDDING_OPENAI_BASE_URL（须为兼容 OpenAI Embeddings 的网关根路径，例如 https://api.openai.com/v1）"
	ConfigKnowledgeEmbeddingModelRequired          = "未设置 KB_EMBEDDING_MODEL"
	// KB_EMBEDDING_DIMENSIONS 须显式配置（须与 PG vector(N) 一致）。
	ConfigKnowledgeBaseEmbeddingDimensionsRequired = "未设置 KB_EMBEDDING_DIMENSIONS（须与向量列一致的整数维度，例如 1536）"
	ConfigKnowledgeBaseEmbeddingDimensionsInvalid  = "KB_EMBEDDING_DIMENSIONS 无效，应为与向量列一致的整数维度（例如 1536）"

	// ConfigKnowledgeEmbeddingGatewayAPIKeyRequired 知识库向量专用网关未配置（勿复用 Moonshot 聊天 Key 调 Embeddings）。
	ConfigKnowledgeEmbeddingGatewayAPIKeyRequired = "未设置 KB_EMBEDDING_OPENAI_API_KEY（知识库向量 Embeddings 专用；Moonshot 不提供 POST /v1/embeddings）"
	// ConfigKnowledgeEmbeddingBatchSizeInvalid 单次 Embeddings 请求条数；阿里云 DashScope 等上限为 10，OpenAI 可设更大。
	ConfigKnowledgeEmbeddingBatchSizeInvalid = "KB_EMBEDDING_BATCH_SIZE 无效，应为 1～2048 的整数（留空则默认 10，兼容 DashScope 单批上限）"

	// 知识库 AI 分片（Chat，与 KB_EMBEDDING_* 网关无关）。
	ConfigKnowledgeChunkMaxInputRunesInvalid       = "KB_CHUNK_MAX_INPUT_RUNES 无效，须为不小于 1024 的整数（留空则沿用 RESUME_AI_MAX_RUNES）"
	ConfigKnowledgeChunkMaxCompletionTokensInvalid = "KB_CHUNK_MAX_COMPLETION_TOKENS 无效，须为 1024～262144 的整数（留空则默认 32768；Moonshot 要求 prompt_tokens+max_completion≤256K）"
	ConfigKnowledgeChunkTemperatureInvalid         = "KB_CHUNK_TEMPERATURE 无效，须为 0～2 的数（留空则沿用 RESUME_AI_TEMPERATURE；Moonshot 部分模型仅允许 1）"
)
