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
	AuthSecret     string
	Logger         *slog.Logger
	Env            string
	RequestTimeout time.Duration

	// P3 业务端口；为 nil 时对应路由不挂载（便于单测只测部分模块）。
	Catalog app.CatalogStore
	Points  app.PointsStore
	Orders  app.OrderStore

	// M8 运营后台；Roles 用于校验管理员身份。
	Admin app.AdminStore
	Roles app.RoleStore
}

// NewRouter 组装全部 HTTP 路由。
//
// 已实现认证、行程、商品、积分、订单和运营后台路由。
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

	auth := NewAuthHandler(opt.DB, opt.AuthSecret, logger)
	catalog := catalogHandler{store: opt.Catalog, logger: logger}
	points := pointsHandler{store: opt.Points, logger: logger}
	orders := orderHandler{store: opt.Orders, logger: logger}
	admin := adminHandler{store: opt.Admin, logger: logger}
	trips := NewTripHandler(opt.DB, logger)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			OK(w, map[string]string{"pong": "green-travel-points", "env": opt.Env})
		})
		r.Post("/auth/register", auth.Register)
		r.Post("/auth/login", auth.Login)
		if opt.Catalog != nil {
			r.Get("/products", catalog.list)
			r.Get("/products/{id}", catalog.get)
		}
		r.Group(func(r chi.Router) {
			r.Use(auth.Require)
			r.Get("/me", auth.Me)
			trips.Routes(r)
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

		if opt.Admin != nil && opt.Roles != nil {
			r.Group(func(r chi.Router) {
				r.Use(auth.Require)
				r.Use(RequireAdmin(opt.Roles, logger))
				r.Get("/admin/products", admin.listProducts)
				r.Post("/admin/products", admin.createProduct)
				r.Patch("/admin/products/{id}", admin.updateProduct)
				r.Put("/admin/skus/{id}/stock", admin.setStock)
				r.Get("/admin/orders", admin.listOrders)
				r.Post("/admin/orders/{id}/confirm", admin.transition("confirm", "PAID"))
				r.Post("/admin/orders/{id}/ship", admin.transition("ship", "SHIPPED"))
				r.Post("/admin/orders/{id}/complete", admin.transition("complete", "COMPLETED"))
				r.Get("/admin/points/reconcile", admin.reconcile)
			})
		}
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Fail(w, http.StatusNotFound, CodeNotFound, "接口不存在: "+r.URL.Path)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Fail(w, http.StatusMethodNotAllowed, CodeInvalidArgument, "方法不允许: "+r.Method)
	})
	return r
}
