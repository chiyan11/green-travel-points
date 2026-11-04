package httpserver

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Options 是构造路由所需的依赖。
type Options struct {
	DB             *sql.DB
	AuthSecret     string
	Logger         *slog.Logger
	Env            string
	RequestTimeout time.Duration
}

// NewRouter 组装全部 HTTP 路由。
//
// P2 已挂载认证与行程链路，其余业务路由按 Sprint 继续挂载：
//
//	/api/v1/auth     用户与认证        (M1, P2)
//	/api/v1/trips    行程上报与碳核算  (M2, P2)
//	/api/v1/points   积分账户与流水    (M3/M5, P3)
//	/api/v1/products 商品与库存        (M6, P3)
//	/api/v1/orders   兑换与订单状态机  (M7, P3)
//	/api/v1/admin    运营后台          (M8, P5)
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
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			OK(w, map[string]string{"pong": "green-travel-points", "env": opt.Env})
		})
		r.Post("/auth/register", auth.Register)
		r.Post("/auth/login", auth.Login)
		r.Group(func(r chi.Router) {
			r.Use(auth.Require)
			r.Get("/me", auth.Me)
			NewTripHandler(opt.DB, logger).Routes(r)
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
