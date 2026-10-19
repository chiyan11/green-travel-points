package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testRouter(dbless bool) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(Options{
		DB:             nil, // Sprint 0 的冒烟测试不依赖 MySQL
		Logger:         logger,
		Env:            "test",
		RequestTimeout: 2 * time.Second,
	})
}

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	testRouter(true).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", rr.Code)
	}
	var env Envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if env.Code != CodeOK {
		t.Fatalf("code = %q, 期望 %q", env.Code, CodeOK)
	}
}

func TestReadyzWithoutDatabase(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	testRouter(true).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("无数据库时就绪探针应为 503，实际 %d", rr.Code)
	}
	var env Envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if env.Code != CodeUnavailable {
		t.Fatalf("code = %q, 期望 %q", env.Code, CodeUnavailable)
	}
}

func TestPing(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	testRouter(true).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestNotFoundUsesEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil)
	testRouter(true).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("状态码 = %d, 期望 404", rr.Code)
	}
	var env Envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if env.Code != CodeNotFound {
		t.Fatalf("code = %q, 期望 %q", env.Code, CodeNotFound)
	}
}
