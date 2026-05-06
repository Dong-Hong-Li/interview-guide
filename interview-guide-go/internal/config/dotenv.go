package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"interview-guide-go/shared/errmsg"
)

// LoadDotEnvOptional 在非空 ENV_FILE 时从该路径加载 dotenv（KEY=VALUE，# 注释）；
// 不覆盖已为进程设置的变量，可与 docker compose 的 env_file 注入并存。
func LoadDotEnvOptional() error {
	path := strings.TrimSpace(os.Getenv("ENV_FILE"))
	if path == "" {
		return nil
	}
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("%s: 读取 ENV_FILE=%q 失败（请确认已挂载宿主 .env）: %w", errmsg.LogFatalLoadEnvFile, path, err)
	}
	return nil
}
