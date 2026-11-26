package repo

import (
	"testing"
	"time"
)

func TestStartOfDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.November, 25, 23, 59, 59, 0, loc)
	want := time.Date(2026, time.November, 25, 0, 0, 0, 0, loc)
	if got := startOfDay(now); !got.Equal(want) || got.Location() != loc {
		t.Fatalf("startOfDay() = %v (%v), want %v (%v)", got, got.Location(), want, loc)
	}
}
