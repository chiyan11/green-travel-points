package repo

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"greentravelpoints/internal/app"
)

// GrantPointsWithBatch 在调用方事务内为一次积分发放写入 point_batch。
//
// 供行程上报（P2）复用：调用方先 INSERT point_txn 并拿到 txnID，
// 再调用本函数；expiresAt 通常为发放时间 + sys_config.points_expire_days。
// 这样 point_account.balance 与 point_batch.remaining 才能保持严格相等。
func GrantPointsWithBatch(ctx context.Context, tx *sql.Tx, accountID, txnID, amount int64, expiresAt time.Time) error {
	if amount <= 0 {
		return nil
	}
	return insertBatch(ctx, tx, accountID, txnID, amount, expiresAt)
}

func insertBatch(ctx context.Context, tx *sql.Tx, accountID, txnID, amount int64, expiresAt time.Time) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO point_batch (account_id, grant_txn_id, granted_amount, remaining, expires_at, status)
		 VALUES (?, ?, ?, ?, ?, 'ACTIVE')`,
		accountID, txnID, amount, amount, expiresAt)
	return err
}

// consumeBatches 按 expires_at、id 升序（真 FIFO）扣减批次，
// 同时把"哪笔流水消耗了哪个批次多少"写入 point_batch_txn，
// 供取消订单时原路回填。
func consumeBatches(ctx context.Context, tx *sql.Tx, accountID, need, txnID int64) error {
	if need <= 0 {
		return nil
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT id, remaining FROM point_batch
		 WHERE account_id = ? AND status = 'ACTIVE' AND remaining > 0
		 ORDER BY expires_at ASC, id ASC FOR UPDATE`, accountID)
	if err != nil {
		return err
	}
	type batch struct{ id, remaining int64 }
	batches := make([]batch, 0, 8)
	for rows.Next() {
		var b batch
		if err := rows.Scan(&b.id, &b.remaining); err != nil {
			_ = rows.Close()
			return err
		}
		batches = append(batches, b)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	var available int64
	for _, b := range batches {
		available += b.remaining
	}
	if available < need {
		return app.ErrInsufficientPoints
	}

	left := need
	for _, b := range batches {
		if left <= 0 {
			break
		}
		take := b.remaining
		if take > left {
			take = left
		}
		left -= take
		remaining := b.remaining - take
		status := "ACTIVE"
		if remaining == 0 {
			status = "CONSUMED"
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE point_batch SET remaining = ?, status = ? WHERE id = ?`,
			remaining, status, b.id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO point_batch_txn (batch_id, txn_id, amount) VALUES (?, ?, ?)`,
			b.id, txnID, -take); err != nil {
			return err
		}
	}
	return nil
}

// restoreBatches 把一笔兑换（REDEEM）消耗过的批次原路回填（expires_at 不变），
// 返回实际回填的积分数量；找不到消费明细时返回 0，由调用方决定是否新建批次。
func restoreBatches(ctx context.Context, tx *sql.Tx, orderID, refundTxnID, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	var redeemTxnID int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM point_txn
		 WHERE biz_ref_type = 'redeem_order' AND biz_ref_id = ? AND txn_type = 'REDEEM'
		 ORDER BY id DESC LIMIT 1`, orderID).Scan(&redeemTxnID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	rows, err := tx.QueryContext(ctx,
		`SELECT batch_id, -amount FROM point_batch_txn
		 WHERE txn_id = ? AND amount < 0 ORDER BY batch_id ASC`, redeemTxnID)
	if err != nil {
		return 0, err
	}
	type consumption struct{ batchID, qty int64 }
	items := make([]consumption, 0, 4)
	for rows.Next() {
		var it consumption
		if err := rows.Scan(&it.batchID, &it.qty); err != nil {
			_ = rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	_ = rows.Close()

	// 固定按 batch_id 升序回填，避免与其他事务交叉锁顺序。
	sort.Slice(items, func(i, j int) bool { return items[i].batchID < items[j].batchID })

	restored := int64(0)
	for _, it := range items {
		qty := it.qty
		if restored+qty > amount {
			qty = amount - restored
		}
		if qty <= 0 {
			break
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE point_batch SET remaining = remaining + ?, status = 'ACTIVE' WHERE id = ?`,
			qty, it.batchID); err != nil {
			return restored, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO point_batch_txn (batch_id, txn_id, amount) VALUES (?, ?, ?)`,
			it.batchID, refundTxnID, qty); err != nil {
			return restored, err
		}
		restored += qty
	}
	return restored, nil
}
