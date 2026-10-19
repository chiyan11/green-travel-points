package httpserver

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"
)

// Health 汇总探针所需依赖。
type Health struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Liveness 只表示进程活着，永远 200；用于容器重启判定。
func (h Health) Liveness(w http.ResponseWriter, r *http.Request) {
	OK(w, map[string]string{"status": "ok"})
}

// Readiness 表示"能否对外提供完整服务"，依赖 MySQL 可 ping 通；
// 数据库未就绪时返回 503，负载均衡/编排据此摘流量。
func (h Health) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库未初始化")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.DB.PingContext(ctx); err != nil {
		h.Logger.Warn("readiness: 数据库不可用", "err", err)
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	OK(w, map[string]string{"status": "ok", "database": "up"})
}
