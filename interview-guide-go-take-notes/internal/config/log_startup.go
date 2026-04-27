package config

import (
	"encoding/json"
	"net/url"

	"go.uber.org/zap"

	"interview-guide-go/shared/errmsg"
)

// maskSecret 非空则打码，避免日志泄露密钥。
func maskSecret(s string) string {
	if s == "" {
		return "(empty)"
	}
	return "***"
}

// redactPostgresURL 尽量保留 host/db，隐藏 userinfo 中的密码。
func redactPostgresURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	name := u.User.Username()
	if _, hasPass := u.User.Password(); hasPass {
		u.User = url.UserPassword(name, "***")
	}
	return u.String()
}

// startupSnapshot 仅用于启动时人类可读、脱敏后的 JSON 输出。
func (c *Config) startupSnapshot() map[string]any {
	o := c.Openai
	return map[string]any{
		"server": c.Server,
		"postgres": map[string]any{
			"database_url": redactPostgresURL(c.Database.DatabaseURL),
			"host":         c.Database.PGHost,
			"port":         c.Database.PGPort,
			"user":         c.Database.PGUser,
			"password":     maskSecret(c.Database.PGPassword),
			"database":     c.Database.PGDBName,
			"sslmode":      c.Database.PGSSLMode,
		},
		"redis": map[string]any{
			"host":     c.Redis.RedisHost,
			"port":     c.Redis.RedisPort,
			"db":       c.Redis.RedisDB,
			"password": maskSecret(c.Redis.RedisPassword),
		},
		"storage": map[string]any{
			"endpoint":                c.Storage.StorageEndpoint,
			"access_key":              maskSecret(c.Storage.StorageAccessKey),
			"secret_key":              maskSecret(c.Storage.StorageSecretKey),
			"bucket":                  c.Storage.StorageBucket,
			"region":                  c.Storage.StorageRegion,
			"presign_get_expires_sec": c.Storage.StoragePresignGetExpiresSec,
		},
		"openai": map[string]any{
			"base_url":                        o.OpenAIBaseURL,
			"api_key":                         maskSecret(o.OpenAIAPIKey),
			"moonshot_api_key":                maskSecret(o.MoonshotAPIKey),
			"model":                           o.AIModel,
			"resume_ai_max_runes":             o.ResumeAIMaxRunes,
			"resume_ai_max_completion_tokens": o.ResumeAIMaxCompletionTokens,
			"resume_ai_temperature":           o.ResumeAITemperature,
		},
		"http_access_log_suppress": c.HTTPAccessLogSuppress,
	}
}

// LogStartup 在进程启动时输出一份配置快照（敏感字段已脱敏）。
// 使用缩进 JSON 放进一条 Info 的 message，避免 ConsoleEncoder 把所有 Field 压成一行紧凑 JSON。
func (c *Config) LogStartup(lg *zap.Logger) {
	if lg == nil {
		return
	}
	b, err := json.MarshalIndent(c.startupSnapshot(), "", "  ")
	if err != nil {
		lg.Info(errmsg.ConfigStartupSnapshotMarshalFail, zap.Error(err))
		return
	}
	lg.Info("startup configuration\n" + string(b))
}
