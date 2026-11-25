package httpserver

import (
	"testing"
	"time"
)

func TestValidateTraveledAt(t *testing.T) {
	now := time.Date(2026, time.November, 25, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"当前时刻", now, false},
		{"七天边界", now.AddDate(0, 0, -7), false},
		{"未来时刻", now.Add(time.Nanosecond), true},
		{"早于七天", now.AddDate(0, 0, -7).Add(-time.Nanosecond), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateTraveledAt(tt.at, now) != nil; got != tt.want {
				t.Fatalf("validateTraveledAt() error = %v, want error = %v", got, tt.want)
			}
		})
	}
}

func TestStartOfDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.November, 25, 23, 59, 59, 0, loc)
	want := time.Date(2026, time.November, 25, 0, 0, 0, 0, loc)
	if got := startOfDay(now); !got.Equal(want) || got.Location() != loc {
		t.Fatalf("startOfDay() = %v (%v), want %v (%v)", got, got.Location(), want, loc)
	}
}
