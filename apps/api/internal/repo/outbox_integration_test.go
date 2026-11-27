package repo

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"greentravelpoints/internal/eventbus"
)

func TestOutboxDispatchRetryAndDeadLetter(t *testing.T) {
	store := newTestStore(t)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	ctx := context.Background()
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES ('TestEvent', 'test', 'retry', '{}')`)
	if err != nil {
		t.Fatal(err)
	}
	outbox := NewOutbox(store.db)

	if _, err := outbox.Dispatch(ctx, 10, 3, func(eventbus.Event) error {
		return errors.New("temporary failure")
	}); err != nil {
		t.Fatal(err)
	}
	var attempts int
	var dispatchedAt, deadLetterAt sql.NullTime
	if err := store.db.QueryRowContext(ctx, `
		SELECT attempts, dispatched_at, dead_letter_at FROM event_outbox WHERE aggregate_id = 'retry'
	`).Scan(&attempts, &dispatchedAt, &deadLetterAt); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || dispatchedAt.Valid || deadLetterAt.Valid {
		t.Fatalf("after retry: attempts=%d dispatchedAt=%v deadLetterAt=%v", attempts, dispatchedAt, deadLetterAt)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE event_outbox SET available_at = NOW(3) WHERE aggregate_id = 'retry'`); err != nil {
		t.Fatal(err)
	}
	if _, err := outbox.Dispatch(ctx, 10, 3, func(eventbus.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `
		SELECT attempts, dispatched_at, dead_letter_at FROM event_outbox WHERE aggregate_id = 'retry'
	`).Scan(&attempts, &dispatchedAt, &deadLetterAt); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || !dispatchedAt.Valid || deadLetterAt.Valid {
		t.Fatalf("after successful replay: attempts=%d dispatchedAt=%v deadLetterAt=%v", attempts, dispatchedAt, deadLetterAt)
	}

	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES ('TestEvent', 'test', 'dead', '{}')`); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := outbox.Dispatch(ctx, 10, 3, func(eventbus.Event) error {
			return errors.New("permanent failure")
		}); err != nil {
			t.Fatal(err)
		}
		if attempt < 3 {
			if _, err := store.db.ExecContext(ctx, `UPDATE event_outbox SET available_at = NOW(3) WHERE aggregate_id = 'dead'`); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := store.db.QueryRowContext(ctx, `
		SELECT attempts, dispatched_at, dead_letter_at FROM event_outbox WHERE aggregate_id = 'dead'
	`).Scan(&attempts, &dispatchedAt, &deadLetterAt); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || dispatchedAt.Valid || !deadLetterAt.Valid {
		t.Fatalf("after max attempts: attempts=%d dispatchedAt=%v deadLetterAt=%v", attempts, dispatchedAt, deadLetterAt)
	}
	if !strings.Contains(logs.String(), `"msg":"outbox event entered dead letter"`) ||
		!strings.Contains(logs.String(), `"topic":"TestEvent"`) ||
		!strings.Contains(logs.String(), `"attempts":3`) ||
		!strings.Contains(logs.String(), `"event_id":`) {
		t.Fatalf("missing dead-letter alert: %s", logs.String())
	}
}

func TestOutboxHandlerRunsWithoutRowLock(t *testing.T) {
	store := newTestStore(t)
	mustExec(t, store.db, `INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES ('TestEvent', 'test', 'unlocked', '{}')`)
	outbox := NewOutbox(store.db)
	count, err := outbox.Dispatch(context.Background(), 1, 3, func(e eventbus.Event) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := store.db.ExecContext(ctx, "UPDATE event_outbox SET last_error = 'checked' WHERE id = ?", e.ID); err != nil {
			return err
		}
		n, err := outbox.Dispatch(ctx, 1, 3, func(eventbus.Event) error {
			t.Error("active lease was dispatched twice")
			return nil
		})
		if err != nil || n != 0 {
			t.Errorf("concurrent dispatch = %d, %v", n, err)
		}
		return nil
	})
	if err != nil || count != 1 {
		t.Fatalf("dispatch = %d, %v", count, err)
	}
	if got := mustQueryInt(t, store.db, "SELECT COUNT(*) FROM event_outbox WHERE aggregate_id = 'unlocked' AND dispatched_at IS NOT NULL"); got != 1 {
		t.Fatalf("dispatched count = %d", got)
	}
}

func TestOutboxExpiredLeaseCannotOverwriteNewOwner(t *testing.T) {
	store := newTestStore(t)
	mustExec(t, store.db, `INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES ('TestEvent', 'test', 'lease', '{}')`)
	outbox := NewOutbox(store.db)
	ctx := context.Background()
	first, oldToken, err := outbox.claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, store.db, "UPDATE event_outbox SET lease_until = DATE_SUB(NOW(3), INTERVAL 1 SECOND) WHERE id = ?", first.ID)
	second, newToken, err := outbox.claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || oldToken == newToken {
		t.Fatal("expired lease was not reclaimed with a new token")
	}
	if err := outbox.finish(ctx, first, oldToken, 3, nil); err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("stale completion error = %v", err)
	}
	if err := outbox.finish(ctx, second, newToken, 3, nil); err != nil {
		t.Fatal(err)
	}
}
