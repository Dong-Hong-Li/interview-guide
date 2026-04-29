package middleware

import (
	"net/http"

	"interview-guide-go/internal/config"

	"github.com/go-chi/cors"
)

// CORS 根据 config 注册 go-chi/cors；应放在路由树最前若干层，使 OPTIONS 预检与跨域 GET（如下载文件）均带 Access-Control-*。
// 须在 LoadEnvironmentVariables 已校验 CorsAllowedOrigins 非空。
func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	if cfg == nil {
		panic("interview-guide-go: config required for CORS middleware")
	}
	opts := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-Id"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if len(cfg.CorsAllowedOrigins) == 1 && cfg.CorsAllowedOrigins[0] == "*" {
		opts.AllowedOrigins = []string{"*"}
		opts.AllowCredentials = false
	} else {
		opts.AllowedOrigins = cfg.CorsAllowedOrigins
	}
	return cors.Handler(opts)
}
