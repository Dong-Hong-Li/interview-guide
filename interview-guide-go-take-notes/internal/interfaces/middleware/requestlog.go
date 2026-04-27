package middleware

import (
	"interview-guide-go/internal/config"
	"net/http"
	"path"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RequestLogger 访问日志；对所有经过该链的 modules 路由生效。
// suppress 为 nil 或空时不屏蔽；规则见 ParseAccessLogSuppressSpec（环境变量 HTTP_ACCESS_LOG_SUPPRESS）。
func RequestLogger(lg *zap.Logger, suppress []config.AccessLogSuppressRule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			if ShouldSuppressAccessLog(suppress, r.Method, r.URL.Path) {
				return
			}
			lg.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// ShouldSuppressAccessLog 若命中任一条规则则不再打 INFO 访问日志。
func ShouldSuppressAccessLog(rules []config.AccessLogSuppressRule, method, urlPath string) bool {
	if len(rules) == 0 {
		return false
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	cp := strings.TrimSpace(urlPath)
	if cp == "" {
		cp = "/"
	} else {
		cp = path.Clean(cp)
		if cp == "." {
			cp = "/"
		}
	}
	for _, ru := range rules {
		if ru.Method != "*" && ru.Method != method {
			continue
		}
		ok, err := path.Match(ru.Pattern, cp)
		if err != nil {
			continue
		}
		if ok {
			return true
		}
	}
	return false
}
