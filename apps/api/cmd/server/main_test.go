package main

import (
	"testing"
	"time"
)

func TestServerWriteTimeoutFollowsRequestTimeout(t *testing.T) {
	cases := []struct {
		name           string
		requestTimeout time.Duration
		want           time.Duration
	}{
		{name: "默认请求超时 15s 时写超时 30s，与改造前一致", requestTimeout: 15 * time.Second, want: 30 * time.Second},
		{name: "共享库场景把请求超时抬到 90s，写超时必须跟着抬高", requestTimeout: 90 * time.Second, want: 105 * time.Second},
		{name: "零值回落默认 15s 再加余量", requestTimeout: 0, want: 30 * time.Second},
		{name: "负值同样回落默认", requestTimeout: -time.Second, want: 30 * time.Second},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := serverWriteTimeout(c.requestTimeout)
			if got != c.want {
				t.Fatalf("serverWriteTimeout(%v) = %v, want %v", c.requestTimeout, got, c.want)
			}
			if c.requestTimeout > 0 && got <= c.requestTimeout {
				t.Fatalf("写超时 %v 必须晚于请求超时 %v，否则慢请求会被服务器提前掐断", got, c.requestTimeout)
			}
		})
	}
}
