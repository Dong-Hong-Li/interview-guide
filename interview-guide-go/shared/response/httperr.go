package response

import (
	"errors"
	"interview-guide-go/shared/errmsg"
	"net/http"
)

// Error 实现 error，并携带与前端约定一致的 HTTP 状态码（写入 Result.code）。
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Err 构造带状态码的 HTTP 错误，供业务层 return，由 handler 包装器或 WriteErr 统一写出。
func Err(code int, message string) error {
	return &Error{Code: code, Message: message}
}

// ErrJSON 写出与项目统一的 Result 错误体（HTTP 状态码与 body.code 一致）。
func ErrJSON(w http.ResponseWriter, code int, message string) {
	WriteJSON(w, code, Result{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// WriteErr 若 err 为 *Error 则按其 Code/Message 写出；否则 500 + 统一文案。
func WriteErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var he *Error
	if errors.As(err, &he) {
		ErrJSON(w, he.Code, he.Message)
		return
	}
	ErrJSON(w, http.StatusInternalServerError, errmsg.InternalServerError)
}

// BizError 表示业务级错误：HTTP 状态码为 200，但 Result.code 为自定义业务码。
type BizError struct {
	Code    int
	Message string
}

func (e *BizError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// BizErr 构造业务级错误。
func BizErr(code int, message string) error {
	return &BizError{Code: code, Message: message}
}
