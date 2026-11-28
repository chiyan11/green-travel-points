package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type timeoutWriter struct {
	http.ResponseWriter
	ctx       context.Context
	committed bool
	timedOut  bool
}

func (w *timeoutWriter) WriteHeader(status int) {
	if w.committed {
		return
	}
	w.committed = true
	if errors.Is(w.ctx.Err(), context.DeadlineExceeded) {
		w.timedOut = true
		Fail(w.ResponseWriter, http.StatusGatewayTimeout, CodeUnavailable, "请求超时")
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *timeoutWriter) Write(p []byte) (int, error) {
	if !w.committed {
		w.WriteHeader(http.StatusOK)
	}
	if w.timedOut {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}

func timeoutJSON(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			tw := &timeoutWriter{ResponseWriter: w, ctx: ctx}
			next.ServeHTTP(tw, r.WithContext(ctx))
			if !tw.committed && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				tw.WriteHeader(http.StatusGatewayTimeout)
			}
		})
	}
}

// statusRecorder 记录响应状态码与字节数，供访问日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// RequestLogger 输出结构化访问日志（method / path / status / 耗时 / 请求 ID）。
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			logger.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
				"remote_ip", r.RemoteAddr,
			)
		})
	}
}
