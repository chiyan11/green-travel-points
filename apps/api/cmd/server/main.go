// Command server 是绿色出行积分系统的 HTTP 入口。
//
// 启动： go run ./cmd/server
// 探针： GET /healthz（存活）、GET /readyz（就绪，含 MySQL ping）
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"greentravelpoints/internal/config"
	"greentravelpoints/internal/httpserver"
	"greentravelpoints/internal/repo"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务退出", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(cfg.Env)
	slog.SetDefault(logger)

	// 启动时播报数据库目标：连远程共享实例时，本机的读写对其他成员实时可见，避免连错库。
	if addr, local, ok := dbEndpoint(cfg.DatabaseDSN); !ok {
		logger.Warn("无法解析 DB_DSN，跳过数据库目标提示")
	} else if local {
		logger.Info("数据库目标：本机实例", "db_addr", addr)
	} else {
		logger.Warn("数据库目标：远程共享实例，你的读写对其他成员实时可见", "db_addr", addr)
	}

	db, err := repo.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warn("关闭数据库连接池失败", "err", err)
		}
	}()

	// 注意：这里不因数据库不可用而退出，方便只调后端时先跑起来。
	// 是否真正可用由 /readyz 暴露，部署时用它做探针。
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	if err := db.PingContext(pingCtx); err != nil {
		logger.Warn("启动时数据库不可用，服务继续运行（/readyz 将返回 503）", "err", err)
	} else {
		logger.Info("数据库连接正常")
	}
	cancelPing()

	// P3 业务：商品目录、积分查询、兑换订单共用同一个 Store。
	store := repo.New(db)

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpserver.NewRouter(httpserver.Options{
			DB:             db,
			AuthSecret:     cfg.AuthSecret,
			Logger:         logger,
			Env:            cfg.Env,
			RequestTimeout: cfg.HTTPRequestTimeout,
			Catalog:        store,
			Points:         store,
			Orders:         store,
			Admin:          store,
			Roles:          store,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// 写超时必须在业务请求超时之后到期，否则 middleware.Timeout 还没到，
		// 服务器就先把连接掐断：客户端（含 Vite dev 代理）看到 502，
		// 后端日志里只剩一条 duration≈30s 的成功记录。跨机共享库上单个写请求
		// 动辄 20–40s，这条关系是硬要求。
		WriteTimeout: serverWriteTimeout(cfg.HTTPRequestTimeout),
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("HTTP 服务启动", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅关闭", "timeout", cfg.HTTPShutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.Info("已优雅关闭")
		return nil
	case err := <-errCh:
		return err
	}
}

func newLogger(env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if env == "dev" {
		opts.Level = slog.LevelDebug
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// serverWriteTimeout 让 http.Server 的写超时始终晚于业务请求超时到期。
//
// middleware.Timeout(RequestTimeout) 只是给 handler 的 ctx 加上截止时间，真正把
// 响应写回连接的是 http.Server；两者的关系反了（WriteTimeout 更短）就会出现
// 「handler 还在等数据库，服务器已经关连接」——客户端看到 502，日志里却是成功。
func serverWriteTimeout(requestTimeout time.Duration) time.Duration {
	if requestTimeout <= 0 {
		requestTimeout = 15 * time.Second
	}
	return requestTimeout + 15*time.Second
}
