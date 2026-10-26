package httpserver

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"greentravelpoints/internal/app"
)

// Options 是构造路由所需的依赖。
type Options struct {
	DB             *sql.DB
	Logger         *slog.Logger
	Env            string
	RequestTimeout time.Duration

	// P3 业务端口；为 nil 时对应路由不挂载（便于单测只测部分模块）。
	Catalog app.CatalogStore
	Points  app.PointsStore
	Orders  app.OrderStore
}

// NewRouter 组装全部 HTTP 路由。
//
// 已实现：
//
//	GET  /healthz /readyz            探针
//	GET  /api/v1/ping                连通性自检
//	GET  /api/v1/products            商品列表（M6，公开）
//	GET  /api/v1/products/{id}       商品详情（M6，公开）
//	GET  /api/v1/points/summary      积分概览（M5，需身份）
//	GET  /api/v1/points/transactions 积分流水（M5，需身份）
//	POST /api/v1/orders              兑换下单（M7，需身份 + Idempotency-Key）
//	GET  /api/v1/orders              我的订单（M7，需身份）
//	POST /api/v1/orders/{id}/cancel  取消订单（M7，需身份）
//
// 待实现：/api/v1/auth、/api/v1/trips（P2）；/api/v1/admin（P5）。
func NewRouter(opt Options) http.Handler {
	if opt.RequestTimeout <= 0 {
		opt.RequestTimeout = 15 * time.Second
	}
	logger := opt.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger(logger))
	r.Use(middleware.Timeout(opt.RequestTimeout))

	health := Health{DB: opt.DB, Logger: logger}
	r.Get("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness)

	catalog := catalogHandler{store: opt.Catalog, logger: logger}
	points := pointsHandler{store: opt.Points, logger: logger}
	orders := orderHandler{store: opt.Orders, logger: logger}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			OK(w, map[string]string{"pong": "green-travel-points", "env": opt.Env})
		})

		// 商品目录不需要登录。
		if opt.Catalog != nil {
			r.Get("/products", catalog.list)
			r.Get("/products/{id}", catalog.get)
		}

		// 需要身份的接口（开发期用 X-User-Id，P2 的 JWT 接入后替换中间件）。
		r.Group(func(r chi.Router) {
			r.Use(RequireUser)
			if opt.Points != nil {
				r.Get("/points/summary", points.summary)
				r.Get("/points/transactions", points.transactions)
			}
			if opt.Orders != nil {
				r.Post("/orders", orders.create)
				r.Get("/orders", orders.list)
				r.Post("/orders/{id}/cancel", orders.cancel)
			}
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Fail(w, http.StatusNotFound, CodeNotFound, "接口不存在: "+r.URL.Path)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Fail(w, http.StatusMethodNotAllowed, CodeInvalidArgument, "方法不允许: "+r.Method)
	})
	return r
}
