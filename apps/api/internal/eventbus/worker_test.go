package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

type fakeDispatcher struct {
	calls     int
	processed int
	err       error
}

func (f *fakeDispatcher) Dispatch(_ context.Context, limit, maxAttempts int, handler Handler) (int, error) {
	f.calls++
	if limit != defaultBatchSize {
		return 0, errors.New("unexpected batch size")
	}
	if maxAttempts != defaultMaxAttempts {
		return 0, errors.New("unexpected max attempts")
	}
	if err := handler(Event{ID: 1, Topic: TopicTripVerified}); err != nil {
		return 0, err
	}
	return f.processed, f.err
}

func TestWorkerProcessOnce(t *testing.T) {
	dispatcher := &fakeDispatcher{processed: 1}
	called := 0
	worker := NewWorker(dispatcher, func(e Event) error {
		called++
		if e.ID != 1 || e.Topic != TopicTripVerified {
			t.Fatalf("unexpected event: %+v", e)
		}
		return nil
	}, slog.Default())

	worker.ProcessOnce(context.Background())

	if dispatcher.calls != 1 {
		t.Fatalf("Dispatch called %d times, want 1", dispatcher.calls)
	}
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
}

func TestWorkerProcessOnceSkipsIncompleteDependencies(t *testing.T) {
	worker := NewWorker(nil, func(Event) error { return nil }, slog.Default())
	worker.ProcessOnce(context.Background())
}
