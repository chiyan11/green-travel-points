package repo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"greentravelpoints/internal/eventbus"
)

func TestOutboxDispatchRetryAndDeadLetter(t *testing.T) {
	store := newTestStore(t)
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
}
