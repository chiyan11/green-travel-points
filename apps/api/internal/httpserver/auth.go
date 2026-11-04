package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"greentravelpoints/internal/app"
)

type ctxKey int

const userIDKey ctxKey = iota

// RequireUser 是**开发期**的临时身份中间件：从请求头 X-User-Id 取用户 ID。
//
// TODO(P2)：M1 认证接口上线后，替换为解析 JWT 并把 userID 放进 context；
// 本中间件与前端接口清单的约定保持独立，业务 Handler 只依赖 UserID(ctx)。
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-User-Id")
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			Fail(w, http.StatusUnauthorized, CodeUnauthorized, "缺少有效的 X-User-Id 请求头（P2 的 JWT 认证尚未接入）")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, id)))
	})
}

// UserID 取出 RequireUser 注入的用户 ID；未注入时返回 0。
func UserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(userIDKey).(int64); ok {
		return v
	}
	return 0
}

// RequireAdmin 必须挂在 RequireUser 之后：校验当前用户角色为 ADMIN。
// 开发期角色来自 user 表 role 字段（P2 的 JWT 接入后可改为从 token claim 读取）。
func RequireAdmin(roles app.RoleStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, err := roles.RoleOf(r.Context(), UserID(r.Context()))
			if err != nil {
				if errors.Is(err, app.ErrNotFound) {
					Fail(w, http.StatusForbidden, CodeForbidden, "需要管理员权限")
					return
				}
				FailInternal(w, logger, err)
				return
			}
			if role != "ADMIN" {
				Fail(w, http.StatusForbidden, CodeForbidden, "需要管理员权限")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// failFromStore 把业务错误统一映射为 HTTP 响应。
func failFromStore(w http.ResponseWriter, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, app.ErrNotFound):
		Fail(w, http.StatusNotFound, CodeNotFound, "资源不存在")
	case errors.Is(err, app.ErrInsufficientPoints):
		Fail(w, http.StatusConflict, CodeInsufficientPoints, "积分余额不足")
	case errors.Is(err, app.ErrOutOfStock):
		Fail(w, http.StatusConflict, CodeOutOfStock, "库存不足")
	case errors.Is(err, app.ErrProductOffline):
		Fail(w, http.StatusConflict, CodeConflict, "商品不存在或已下架")
	case errors.Is(err, app.ErrLimitExceeded):
		Fail(w, http.StatusConflict, CodeLimitExceeded, "超出单用户限兑数量")
	case errors.Is(err, app.ErrIllegalTransition):
		Fail(w, http.StatusConflict, CodeIllegalTransition, "当前订单状态不允许该操作")
	case errors.Is(err, app.ErrInvalidInput):
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "参数不合法")
	default:
		FailInternal(w, logger, err)
	}
}

// parsePaging 解析 page / pageSize，带默认值与上限。
func parsePaging(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
