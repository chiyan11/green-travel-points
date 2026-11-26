package repo

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempts int
		want     int
	}{
		{attempts: 1, want: 1},
		{attempts: 2, want: 2},
		{attempts: 5, want: 16},
		{attempts: 10, want: 300},
	}
	for _, tt := range tests {
		t.Run("attempt", func(t *testing.T) {
			if got := retryDelay(tt.attempts); got != tt.want {
				t.Fatalf("retry delay = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateError(t *testing.T) {
	if got := truncateError(errors.New(strings.Repeat("x", 1025))); len(got) != 1024 {
		t.Fatalf("truncated error length = %d, want 1024", len(got))
	}
	long := errors.New(strings.Repeat("超时", 2000))
	got := truncateError(long)
	if !utf8.ValidString(got) {
		t.Fatalf("截断后不是合法 UTF-8: %q", got)
	}
	if n := len([]rune(got)); n != 1024 {
		t.Fatalf("rune 长度 = %d，期望 1024", n)
	}
	// 按字节切会留下半个字符；这里要求截断结果等于原串的前 1024 个 rune。
	if want := string([]rune(long.Error())[:1024]); got != want {
		t.Fatalf("截断结果与按 rune 前缀不一致")
	}
}

func TestRetryDelayNeverOverflows(t *testing.T) {
	prev := 0
	for attempts := 1; attempts <= 70; attempts++ {
		got := retryDelay(attempts)
		if got < 1 || got > 300 {
			t.Fatalf("attempts=%d 时 delay=%d，应在 1..300", attempts, got)
		}
		if got < prev {
			t.Fatalf("attempts=%d 时 delay 回退：%d < %d", attempts, got, prev)
		}
		prev = got
	}
}
