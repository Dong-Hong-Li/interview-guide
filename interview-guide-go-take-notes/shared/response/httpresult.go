package response

import (
	"encoding/json"
	"net/http"
)

// Result 与 Java 侧 interview.guide.common.result.Result 字段一致，便于后续对接前端。
type Result struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

const CodeSuccess = 200

// Success 业务成功时的统一 Result。
func Success(data any) Result {
	return Result{Code: CodeSuccess, Message: "success", Data: data}
}

// WriteJSON 输出 JSON，供 chi / 标准库 Handler 使用。
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
