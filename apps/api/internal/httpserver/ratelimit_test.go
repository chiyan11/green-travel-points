package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"
)

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
