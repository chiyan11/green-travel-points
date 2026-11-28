package repo

import (
	"context"
	"testing"
	"time"
)

// TestCreateOrderSkipsExpiredBatches 过期批次不得参与扣减：
// 先把早到期批次 9001 置为已过期，兑换 300 分应只扣晚到期批次 9002。
// 修复前 SQL 没有 expires_at 过滤，会先扣 9001，本用例即为回归护栏。
func TestCreateOrderSkipsExpiredBatches(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db,
		`UPDATE point_batch SET expires_at = DATE_SUB(NOW(3), INTERVAL 1 DAY) WHERE id = ?`,
		testBatchOld)

	order, replay, err := store.CreateOrder(ctx, newOrderInput())
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	if replay {
		t.Fatal("首次下单不应是幂等重放")
	}
	if order.PointsSpent != 300 {
		t.Fatalf("扣减积分 = %d，期望 300", order.PointsSpent)
	}

	if got := mustQueryInt(t, store.db,
		`SELECT remaining FROM point_batch WHERE id = ?`, testBatchOld); got != 400 {
		t.Fatalf("过期批次剩余 = %d，期望 400（不得被扣减）", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT remaining FROM point_batch WHERE id = ?`, testBatchNew); got != 300 {
		t.Fatalf("有效批次剩余 = %d，期望 300", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch_txn WHERE batch_id = ?`, testBatchOld); got != 0 {
		t.Fatalf("过期批次消耗明细 = %d，期望 0", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch_txn WHERE batch_id = ? AND amount = -300`, testBatchNew); got != 1 {
		t.Fatalf("有效批次消耗明细 = %d，期望 1", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 700 {
		t.Fatalf("余额 = %d，期望 700", got)
	}
}

func TestGrantPointsWithBatchKeepsLedgerBalanced(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	expiresAt := time.Now().AddDate(0, 0, 30)
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	const amount int64 = 75
	const balanceAfter int64 = 1075
	res, err := tx.ExecContext(ctx,
		`INSERT INTO point_txn (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
		 VALUES (?, 'TRAVEL', ?, ?, 'trip_record', 9004)`, testAccountID, amount, balanceAfter)
	if err != nil {
		t.Fatal(err)
	}
	txnID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE point_account SET balance = ? WHERE id = ?`, balanceAfter, testAccountID); err != nil {
		t.Fatal(err)
	}
	if err := GrantPointsWithBatch(ctx, tx, testAccountID, txnID, amount, expiresAt); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch WHERE account_id = ? AND grant_txn_id = ?
		 AND granted_amount = ? AND remaining = ? AND status = 'ACTIVE'`,
		testAccountID, txnID, amount, amount); got != 1 {
		t.Fatalf("新发放积分批次数 = %d，期望 1", got)
	}
	assertBatchLedgerBalanced(t, store.db)
	if result, err := store.ReconcilePoints(ctx); err != nil || !result.OK {
		t.Fatalf("发放后积分对账不一致: %+v, err=%v", result, err)
	}
}
