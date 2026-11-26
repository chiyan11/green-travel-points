package repo

import (
	"context"
	"database/sql"
	"fmt"

	"greentravelpoints/internal/eventbus"
)

// Outbox 是 event_outbox 的消费端存储实现。
type Outbox struct {
	db *sql.DB
}

func NewOutbox(db *sql.DB) *Outbox { return &Outbox{db: db} }

// Dispatch 在一个事务中锁定并处理一批事件。处理器成功才标记已投递；失败按指数退避重试，
// 达到上限后写入死信。事务在处理期间持有行锁，避免并发 Worker 重复处理同一行。
func (s *Outbox) Dispatch(ctx context.Context, limit, maxAttempts int, handler eventbus.Handler) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, topic, aggregate_type, aggregate_id, payload, attempts, available_at, created_at
		FROM event_outbox
		WHERE dispatched_at IS NULL
		  AND dead_letter_at IS NULL
		  AND available_at <= NOW(3)
		ORDER BY id
		LIMIT ? FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	events := make([]eventbus.Event, 0, limit)
	for rows.Next() {
		var e eventbus.Event
		if err := rows.Scan(&e.ID, &e.Topic, &e.AggregateType, &e.AggregateID, &e.Payload,
			&e.Attempts, &e.AvailableAt, &e.CreatedAt); err != nil {
			return 0, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	for _, e := range events {
		if err := handler(e); err != nil {
			if err := s.recordFailure(ctx, tx, e, maxAttempts, err); err != nil {
				return 0, err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE event_outbox SET dispatched_at = NOW(3), last_error = NULL WHERE id = ?`, e.ID); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(events), nil
}

func (s *Outbox) recordFailure(ctx context.Context, tx *sql.Tx, e eventbus.Event, maxAttempts int, cause error) error {
	attempts := e.Attempts + 1
	if attempts >= maxAttempts {
		_, err := tx.ExecContext(ctx, `
			UPDATE event_outbox
			SET attempts = ?, last_error = ?, dead_letter_at = NOW(3)
			WHERE id = ?`, attempts, truncateError(cause), e.ID)
		return err
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE event_outbox
		SET attempts = ?, last_error = ?, available_at = DATE_ADD(NOW(3), INTERVAL ? SECOND)
		WHERE id = ?`, attempts, truncateError(cause), retryDelay(attempts), e.ID)
	return err
}

// maxRetryDelaySeconds 是重试退避的上限。
const maxRetryDelaySeconds = 300

// retryDelay 返回第 attempts 次失败后的退避秒数（1, 2, 4, ... 封顶 300）。
// 先按指数上限判断再移位，避免 attempts 很大时 1 << (attempts-1) 溢出成负数。
func retryDelay(attempts int) int {
	if attempts <= 1 {
		return 1
	}
	if attempts-1 >= 9 { // 1<<9 = 512 已超过上限，再往上必然封顶
		return maxRetryDelaySeconds
	}
	delay := 1 << (attempts - 1)
	if delay > maxRetryDelaySeconds {
		return maxRetryDelaySeconds
	}
	return delay
}

// maxTruncatedErrorRunes 是写入 last_error 的最大字符数（按 rune 截断，保证不切坏 UTF-8）。
const maxTruncatedErrorRunes = 1024

// truncateError 把错误信息截断到 1024 个字符：按字节切会把多字节字符切成乱码，
// last_error 是 varchar(utf8mb4) 且会被运营后台展示，因此必须按 rune 边界截断。
func truncateError(err error) string {
	message := fmt.Sprint(err)
	runes := []rune(message)
	if len(runes) <= maxTruncatedErrorRunes {
		return message
	}
	return string(runes[:maxTruncatedErrorRunes])
}

var _ eventbus.Dispatcher = (*Outbox)(nil)
