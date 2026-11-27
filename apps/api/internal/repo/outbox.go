package repo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"greentravelpoints/internal/eventbus"
)

// Outbox 是 event_outbox 的消费端存储实现。
type Outbox struct {
	db *sql.DB
}

func NewOutbox(db *sql.DB) *Outbox { return &Outbox{db: db} }

// Dispatch 逐条短事务认领事件，处理器在事务外执行。租约过期后可能重复投递，消费者必须幂等。
func (s *Outbox) Dispatch(ctx context.Context, limit, maxAttempts int, handler eventbus.Handler) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	processed := 0
	for processed < limit {
		e, token, err := s.claim(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return processed, nil
		}
		if err != nil {
			return processed, err
		}
		cause := handler(e)
		if err := s.finish(ctx, e, token, maxAttempts, cause); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Outbox) claim(ctx context.Context) (eventbus.Event, string, error) {
	var e eventbus.Event
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return e, "", err
	}
	token := hex.EncodeToString(random[:])
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e, "", err
	}
	defer func() { _ = tx.Rollback() }()
	err = tx.QueryRowContext(ctx, `
		SELECT id, topic, aggregate_type, aggregate_id, payload, attempts, available_at, created_at
		FROM event_outbox
		WHERE dispatched_at IS NULL
		  AND dead_letter_at IS NULL
		  AND available_at <= NOW(3)
		  AND (lease_until IS NULL OR lease_until <= NOW(3))
		ORDER BY id
		LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&e.ID, &e.Topic, &e.AggregateType, &e.AggregateID,
		&e.Payload, &e.Attempts, &e.AvailableAt, &e.CreatedAt)
	if err != nil {
		return e, "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_outbox
		SET lease_token = ?, lease_until = DATE_ADD(NOW(3), INTERVAL 5 MINUTE) WHERE id = ?`,
		token, e.ID); err != nil {
		return e, "", err
	}
	if err := tx.Commit(); err != nil {
		return e, "", err
	}
	return e, token, nil
}

func (s *Outbox) finish(ctx context.Context, e eventbus.Event, token string, maxAttempts int, cause error) error {
	var result sql.Result
	var err error
	if cause == nil {
		result, err = s.db.ExecContext(ctx, `UPDATE event_outbox
			SET dispatched_at = NOW(3), last_error = NULL, lease_token = NULL, lease_until = NULL
			WHERE id = ? AND lease_token = ?`, e.ID, token)
	} else if attempts := e.Attempts + 1; attempts >= maxAttempts {
		result, err = s.db.ExecContext(ctx, `
			UPDATE event_outbox
			SET attempts = ?, last_error = ?, dead_letter_at = NOW(3), lease_token = NULL, lease_until = NULL
			WHERE id = ? AND lease_token = ?`, attempts, truncateError(cause), e.ID, token)
	} else {
		result, err = s.db.ExecContext(ctx, `
			UPDATE event_outbox
			SET attempts = ?, last_error = ?, available_at = DATE_ADD(NOW(3), INTERVAL ? SECOND),
			    lease_token = NULL, lease_until = NULL
			WHERE id = ? AND lease_token = ?`, attempts, truncateError(cause), retryDelay(attempts), e.ID, token)
	}
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return fmt.Errorf("outbox event %d lease lost", e.ID)
	}
	return nil
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
