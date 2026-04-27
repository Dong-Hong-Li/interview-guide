// Package httpserver 站点级 HTTP 壳：全局中间件、/api 挂载、RouteRegistrar 聚合；各业务域的 chi 路由在
// application/<domain>/controller（如 application/resume/controller），由 cmd/server/deps 注入。
package httpserver

import (
	"interview-guide-go/internal/config"
	"interview-guide-go/shared/errmsg"
	"interview-guide-go/shared/response"
	"net/http"

	"interview-guide-go/internal/interfaces/controller/system"
	"interview-guide-go/internal/interfaces/middleware"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// RouteRegistrar 约束各模块控制器在传入的 chi.Router 上注册自己的路由。
type RouteRegistrar interface {
	Register(r chi.Router)
}

// RegistrationRoutes 注册路由
// lg: 日志
// httpAccessLogSuppress: 访问日志抑制规则
// apiControllers: 控制器列表
// return: 路由
func RegistrationRoutes(lg *zap.Logger, httpAccessLogSuppress []config.AccessLogSuppressRule, apiControllers []RouteRegistrar) http.Handler {
	router := chi.NewRouter()
	// 注册全局中间件
	MountGlobal(router, lg, httpAccessLogSuppress)

	// 业务接口统一挂在 /api 下（无 /api/v1 前缀）。
	router.Route("/api", func(r chi.Router) {
		// 挂载处理Hooks
		MountAPIHooks(r)

		// 注册系统级接口（Register 为
		(&system.RootController{}).Register(r)

		for _, m := range apiControllers {
			if m == nil {
				continue
			}
			m.Register(r)
		}
	})

	return router
}

// MountGlobal 最外层入口：挂载整站通用中间件（对所有通过该 mux 的 modules 路由生效；须在业务 Route 之前调用）。
// accessLogSuppress 可为 nil：不屏蔽访问日志；规则语法见 ParseAccessLogSuppressSpec。
// 注册全局中间件
// r: 路由
// lg: 日志
// accessLogSuppress: 访问日志抑制规则
func MountGlobal(r chi.Router, lg *zap.Logger, accessLogSuppress []config.AccessLogSuppressRule) {
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(lg, accessLogSuppress))
}

// MountAPIHooks 对当前路由挂载处理Hooks 比如 405 错误处理,404 错误处理等,
func MountAPIHooks(r chi.Router) {
	r.MethodNotAllowed(APIMethodNotAllowed)
	r.NotFound(APINotFound)
}

// APIMethodNotAllowed 供 /api 下全部业务模块共用：HTTP 方法不允许时统一 405 + Result JSON。
func APIMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	response.ErrJSON(w, http.StatusMethodNotAllowed, errmsg.MethodNotAllowed)
}

// APINotFound 供 /api 下全部业务模块共用：HTTP 路由未找到时统一 404 + Result JSON。
func APINotFound(w http.ResponseWriter, r *http.Request) {
	response.ErrJSON(w, http.StatusNotFound, errmsg.RouteNotFound)
}
