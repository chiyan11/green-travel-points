package httpserver

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// 登录与注册的限流参数：同一来源 IP 每分钟 5 次。
// 计数保存在进程内存，单实例部署足够；多实例部署需要换成共享存储。
const (
	authRateLimitPerMinute = 5
	authRateWindow         = time.Minute
)

type rateBucket struct {
	count int
	reset time.Time
}

// rateLimiter 固定窗口计数器：窗口内超过上限即拒绝，窗口结束自动重置。
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string]rateBucket
	limit  int
	window time.Duration
	now    func() time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:   make(map[string]rateBucket),
		limit:  limit,
		window: window,
		now:    time.Now,
	}
}

// allow 记录一次请求，返回是否放行与 Retry-After 秒数。
func (l *rateLimiter) allow(key string) (bool, int) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	// 桶数量只随来源 IP 增长；超过阈值时顺手清掉已过期的，避免长期运行内存膨胀。
	if len(l.hits) > 1024 {
		for k, b := range l.hits {
			if now.After(b.reset) {
				delete(l.hits, k)
			}
		}
	}
	b, ok := l.hits[key]
	if !ok || now.After(b.reset) {
		l.hits[key] = rateBucket{count: 1, reset: now.Add(l.window)}
		return true, 0
	}
	if b.count >= l.limit {
		// 向上取整到秒，且不超过窗口长度。
		remaining := b.reset.Sub(now)
		secs := int(remaining / time.Second)
		if remaining%time.Second != 0 {
			secs++
		}
		if secs < 1 {
			secs = 1
		}
		return false, secs
	}
	b.count++
	l.hits[key] = b
	return true, 0
}

// clientKey 取请求来源 IP。router 已挂 middleware.RealIP，经反向代理时也能拿到真实地址。
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
