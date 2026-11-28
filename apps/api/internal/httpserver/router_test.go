package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"greentravelpoints/internal/app"
)

type timeoutCatalog struct{ fakeCatalog }

func (timeoutCatalog) ListProducts(ctx context.Context, _ app.ProductQuery) (app.Page[app.Product], error) {
	<-ctx.Done()
	return app.Page[app.Product]{}, ctx.Err()
}

func TestRouterTimeoutJSON(t *testing.T) {
	h := NewRouter(Options{Logger: discardLogger(), Catalog: timeoutCatalog{}, RequestTimeout: 10 * time.Millisecond})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products", nil))
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusGatewayTimeout || env.Code != CodeUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTimeoutJSON(t *testing.T) {
	tests := []struct {
		name   string
		cancel bool
		handle http.HandlerFunc
		status int
		code   string
	}{
		{"normal", false, func(w http.ResponseWriter, _ *http.Request) { OK(w, "ok") }, 200, CodeOK},
		{"deadline before write", false, func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
			Fail(w, 500, CodeInternal, "database detail")
		}, 504, CodeUnavailable},
		{"deadline without write", false, func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}, 504, CodeUnavailable},
		{"client cancellation", true, func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
			Fail(w, 500, CodeInternal, "cancelled")
		}, 500, CodeInternal},
		{"already committed", false, func(w http.ResponseWriter, r *http.Request) {
			OK(w, "ok")
			<-r.Context().Done()
		}, 200, CodeOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cancel {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			timeoutJSON(10*time.Millisecond)(tt.handle).ServeHTTP(rec, req)
			var env Envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("invalid JSON: %v; body=%s", err, rec.Body.String())
			}
			if rec.Code != tt.status || env.Code != tt.code {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

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
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data["pong"] != "green-travel-points" {
		t.Fatalf("unexpected ping response: %s", rr.Body.String())
	}
	if _, ok := body.Data["env"]; ok {
		t.Fatalf("ping should not expose env: %s", rr.Body.String())
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

func TestPagingLimit(t *testing.T) {
	for _, tt := range []struct {
		query    string
		wantPage int
		wantSize int
		wantOK   bool
	}{
		{"", 1, 20, true},
		{"?page=-1&pageSize=101", 1, 100, true},
		{"?page=100000&pageSize=1", 100000, 1, true},
		{"?page=100001", 0, 0, false},
		{"?page=999999999999999999999999999", 0, 0, false},
	} {
		t.Run(tt.query, func(t *testing.T) {
			rec := httptest.NewRecorder()
			page, size, ok := parsePaging(rec, httptest.NewRequest(http.MethodGet, "/"+tt.query, nil))
			if page != tt.wantPage || size != tt.wantSize || ok != tt.wantOK {
				t.Fatalf("page=%d size=%d ok=%v", page, size, ok)
			}
			if !ok && (rec.Code != http.StatusBadRequest || !json.Valid(rec.Body.Bytes())) {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPagingLimitOnListRoutes(t *testing.T) {
	h := NewRouter(Options{
		Logger: discardLogger(), AllowDevHeader: true,
		Catalog: fakeCatalog{}, Points: fakePoints{}, Orders: fakeOrders{},
		Trips: fakeTrips{}, Admin: fakeAdmin{}, Roles: fakeRoles{role: "ADMIN"},
	})
	for _, path := range []string{
		"/api/v1/products", "/api/v1/points/transactions", "/api/v1/orders",
		"/api/v1/trips", "/api/v1/admin/products", "/api/v1/admin/orders",
		"/api/v1/admin/audit-logs",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path+"?page=1000000000", nil)
			req.Header.Set("X-User-Id", "1")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			var env Envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
