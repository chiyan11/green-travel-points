package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimitIgnoresForwardedHeaders(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/register"} {
		t.Run(path, func(t *testing.T) {
			h := NewRouter(Options{})
			for i := 0; i < 6; i++ {
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
				req.RemoteAddr = fmt.Sprintf("192.0.2.1:%d", 1000+i)
				if i > 0 {
					req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
					req.Header.Set("X-Real-IP", fmt.Sprintf("203.0.113.%d", i))
				}
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				if i < 5 && rec.Code == http.StatusTooManyRequests {
					t.Fatalf("request %d unexpectedly limited", i+1)
				}
				if i == 5 && (rec.Code != http.StatusTooManyRequests ||
					rec.Header().Get("Retry-After") == "" ||
					!strings.Contains(rec.Body.String(), `"RATE_LIMITED"`)) {
					t.Fatalf("request 6: status=%d retry=%q body=%s", rec.Code, rec.Header().Get("Retry-After"), rec.Body.String())
				}
			}
		})
	}
}

func TestRateLimiterWindow(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	l := newRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("1.2.3.4"); !ok {
			t.Fatalf("第 %d 次请求应放行", i+1)
		}
	}
	ok, retry := l.allow("1.2.3.4")
	if ok {
		t.Fatalf("超过上限应拒绝")
	}
	if retry <= 0 || retry > 60 {
		t.Fatalf("Retry-After = %d，期望 1..60", retry)
	}
	// 其他来源不受影响。
	if ok, _ := l.allow("5.6.7.8"); !ok {
		t.Fatalf("不同来源 IP 不应共享窗口")
	}
	// 窗口结束后重置。
	now = now.Add(61 * time.Second)
	if ok, _ := l.allow("1.2.3.4"); !ok {
		t.Fatalf("窗口结束后应重新放行")
	}
}

func TestClientKey(t *testing.T) {
	cases := map[string]string{
		"192.0.2.1:1234":    "192.0.2.1",
		"[2001:db8::1]:443": "2001:db8::1",
		"192.0.2.2":         "192.0.2.2",
	}
	for remote, want := range cases {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
		r.RemoteAddr = remote
		if got := clientKey(r); got != want {
			t.Fatalf("clientKey(%q) = %q，期望 %q", remote, got, want)
		}
	}
}
