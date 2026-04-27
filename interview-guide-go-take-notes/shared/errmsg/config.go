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

	ConfigOpenAIAPIKeyRequired       = "未设置 OPENAI_API_KEY"
	ConfigOpenAIBaseURLRequired      = "未设置 OPENAI_BASE_URL"
	ConfigMoonshotAPIKeyRequired     = "未设置 MOONSHOT_API_KEY"
	ConfigAIModelRequired            = "未设置 AI_MODEL"
	ConfigResumeAIMaxRunesInvalid    = "RESUME_AI_MAX_RUNES 无效，须为正整数"
	ConfigResumeAIMaxTokensInvalid   = "RESUME_AI_MAX_COMPLETION_TOKENS 无效，须为正整数"
	ConfigResumeAITemperatureInvalid = "RESUME_AI_TEMPERATURE 无效，须为不小于 0 的数"

	ConfigStartupSnapshotMarshalFail = "启动时序列化配置快照失败"
)

// 进程启动阶段 log.Fatalf 前缀（与具体校验错误拼接）。
const (
	LogFatalValidateServerConfig   = "[配置] 校验 server 失败"
	LogFatalValidateDatabaseConfig = "[配置] 校验 database 失败"
	LogFatalValidateRedisConfig    = "[配置] 校验 redis 失败"
	LogFatalValidateStorageConfig  = "[配置] 校验 storage 失败"
	LogFatalValidateOpenAIConfig   = "[配置] 校验 openai 失败"
)

// 配置为空时的错误文案。
const (
	ConfigOpenAIServiceStartFailed = "OpenAI 服务启动失败"
)
