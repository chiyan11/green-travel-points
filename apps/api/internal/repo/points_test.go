package repo

import (
	"context"
	"testing"
)

func TestDailyCarbonYearWindow(t *testing.T) {
	store := newTestStore(t)
	points, err := store.DailyCarbon(context.Background(), testUserID, 365)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 365 {
		t.Fatalf("daily carbon days = %d, want 365", len(points))
	}
}
