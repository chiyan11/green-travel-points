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

	db, err := repo.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Warn("关闭数据库连接池失败", "err", cerr)
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
			Logger:         logger,
			Env:            cfg.Env,
			RequestTimeout: cfg.HTTPRequestTimeout,
			Catalog:        store,
			Points:         store,
			Orders:         store,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
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
