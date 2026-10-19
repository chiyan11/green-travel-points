package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// 统一响应包装：所有接口都返回 {code, message, data}。
// code 是稳定的业务错误码（供前端与接口测试断言），HTTP 状态码表达协议层语义。
const (
	CodeOK              = "OK"
	CodeInvalidArgument = "INVALID_ARGUMENT"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeInternal        = "INTERNAL"
	CodeUnavailable     = "UNAVAILABLE"
)

// Envelope 是全部 HTTP 接口的响应外壳。
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// WriteJSON 写出 JSON 响应。
func WriteJSON(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		slog.Error("httpserver: 写出响应失败", "err", err)
	}
}

// OK 返回 200 + data。
func OK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 返回指定状态码与业务错误码。
func Fail(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, Envelope{Code: code, Message: message})
}

// FailInternal 返回 500，并记录内部错误详情（不外泄给客户端）。
func FailInternal(w http.ResponseWriter, logger *slog.Logger, err error) {
	logger.Error("httpserver: 内部错误", "err", err)
	Fail(w, http.StatusInternalServerError, CodeInternal, "服务器内部错误")
}
