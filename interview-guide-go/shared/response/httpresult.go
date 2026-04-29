package response

import (
	"encoding/json"
	"net/http"
)

// Result 统一 HTTP 业务返回体（code / message / data），供前端按既定结构解析。
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
