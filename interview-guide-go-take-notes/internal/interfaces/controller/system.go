package controller

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// rootController 站点根：GET /、GET /health（与 docs/项目架构.md「system 包」语义对齐）。
type rootController struct{}

func (c *rootController) Register(r chi.Router) {
	r.Get("/", c.root)
	r.Get("/health", c.health)
}

func (rootController) root(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("interview-guide-go\n"))
}

func (rootController) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// apiMetaController GET /api/meta（轻量元数据；不含 PG/Redis 探测，与「辅助口」口径一致）。
type apiMetaController struct{}

func (c *apiMetaController) Register(r chi.Router) {
	r.Get("/meta", c.meta)
}

func (apiMetaController) meta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}
