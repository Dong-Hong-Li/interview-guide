package system

import (
	"interview-guide-go/shared/response"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RootController 注册站点根路径（不含 /api 前缀），如健康检查、首页信息。
type RootController struct{}

func (c *RootController) Register(r chi.Router) {
	r.Get(PathRootHealth, c.health)
	r.Get(PathRootIndex, c.root)
}

// health 健康检查。GET /health
func (c *RootController) health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, response.Success(map[string]string{"status": "ok"}))
}

// root 站点根信息。GET /
func (c *RootController) root(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, response.Success(map[string]string{
		"name":    "interview-guide-go-take-notes",
		"version": "0.0.1",
		"hint":    "见 internal/interfaces/http/binding：Handle = bindRequest + Validate；JSON/Bind=Handle 别名，Exec=无入参",
	}))
}
