package controller

import (
	"net/http"

	"interview-guide-go/internal/config"
	"interview-guide-go/internal/interfaces/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RouteRegistrar 业务 HTTP 适配层在统一 `/api` 下挂载子树（与各域 `*Controller.Register` 一致）。
type RouteRegistrar interface {
	Register(r chi.Router)
}

// RegistrationRoutes 构建站点根路由：全局中间件、根健康检查、/api 下各域与系统端点。
func RegistrationRoutes(lg *zap.Logger, suppress []config.AccessLogSuppressRule, registrars []RouteRegistrar) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	if lg != nil {
		r.Use(middleware.RequestLogger(lg, suppress))
	}

	(&rootController{}).Register(r)

	r.Route("/api", func(api chi.Router) {
		(&apiMetaController{}).Register(api)
		for _, reg := range registrars {
			if reg != nil {
				reg.Register(api)
			}
		}
	})

	return r
}
