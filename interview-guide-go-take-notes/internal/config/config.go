package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"interview-guide-go/shared/errmsg"
)

// Config 应用运行时配置，均从环境变量在 Load 时一次性解析（不在业务包内散落 os.Getenv）。
type Config struct {
	Server ServerConfig

	// database 配置
	Database DatabaseConfig

	// redis 配置
	Redis RedisConfig

	// storage 配置 OSS 或 S3
	Storage StorageConfig

	// openai 配置
	Openai OpenAIConfig

	// HTTPAccessLogSuppress 访问日志抑制规则
	HTTPAccessLogSuppress []AccessLogSuppressRule

	// CorsAllowedOrigins 必填（逗号分隔，见环境变量 CORS_ALLOWED_ORIGINS）；例如本地前端 http://localhost:5173。
	CorsAllowedOrigins []string

	// 简历上传单文件大小上限
	MaxResumeUploadBytes int64
}

// LoadEnvironmentVariables 从环境变量读取全部配置；
func LoadEnvironmentVariables() *Config {
	serverConfig, err := validateServerConfig()
	if err != nil {
		log.Fatalf("%s: %v", errmsg.LogFatalValidateServerConfig, err)
	}
	databaseConfig, err := validateDatabaseConfig()
	if err != nil {
		log.Fatalf("%s: %v", errmsg.LogFatalValidateDatabaseConfig, err)
	}
	redisConfig, err := validateRedisConfig()
	if err != nil {
		log.Fatalf("%s: %v", errmsg.LogFatalValidateRedisConfig, err)
	}
	storageConfig, err := validateStorageConfig()
	if err != nil {
		log.Fatalf("%s: %v", errmsg.LogFatalValidateStorageConfig, err)
	}
	openaiConfig, err := validateOpenAIConfig()
	if err != nil {
		log.Fatalf("%s: %v", errmsg.LogFatalValidateOpenAIConfig, err)
	}
	// HTTP 访问日志屏蔽规则
	httpAccessLogSuppress := parseHTTPAccessLogSuppress()

	corsAllowedOrigins := parseCORSAllowedOrigins()
	if len(corsAllowedOrigins) == 0 {
		log.Fatalf("%s: %s", errmsg.LogFatalValidateCORSConfig, errmsg.ConfigCORSAllowedOriginsRequired)
	}

	// 简历上传单文件大小上限
	maxResumeUploadBytes := parseMaxResumeUploadBytes()
	return &Config{
		Server:                *serverConfig,
		Database:              *databaseConfig,
		Redis:                 *redisConfig,
		Storage:               *storageConfig,
		Openai:                *openaiConfig,
		HTTPAccessLogSuppress: httpAccessLogSuppress,
		CorsAllowedOrigins:    corsAllowedOrigins,
		MaxResumeUploadBytes:  maxResumeUploadBytes,
	}
}

type AccessLogSuppressRule struct {
	Method  string
	Pattern string
}

// HTTP 访问日志屏蔽规则（path.Match）；含面试页轮询、简历详情、知识库列表页三组轮询 GET。
// parseCORSAllowedOrigins 读取 CORS_ALLOWED_ORIGINS（逗号分隔）；至少须有一项（否则 LoadEnvironmentVariables 失败）。
func parseCORSAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseHTTPAccessLogSuppress() []AccessLogSuppressRule {
	accessLogSuppressRules := []AccessLogSuppressRule{
		{Method: "GET", Pattern: "/api/interview/sessions/*"},
		{Method: "GET", Pattern: "/api/resumes/*/detail"},
		// 知识库管理页三路轮询（list / stats / categories）
		{Method: "GET", Pattern: "/api/knowledgebase/list"},
		{Method: "GET", Pattern: "/api/knowledgebase/stats"},
		{Method: "GET", Pattern: "/api/knowledgebase/categories"},
	}
	return accessLogSuppressRules
}

// 简历上传单文件大小上限（环境变量须为十进制整数字节；允许行内 # 注释，如 20971520 # 20MiB）。
func parseMaxResumeUploadBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("MAX_RESUME_UPLOAD_BYTES"))
	if raw == "" {
		log.Fatalf("parse max resume upload bytes failed: MAX_RESUME_UPLOAD_BYTES is empty")
	}
	if i := strings.Index(raw, "#"); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	maxResumeUploadBytes, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || maxResumeUploadBytes <= 0 {
		log.Fatalf("parse max resume upload bytes failed: %v", err)
	}
	return maxResumeUploadBytes
}
