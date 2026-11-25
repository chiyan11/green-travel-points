package repo

import (
	"errors"
	"strings"
	"testing"
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
}
