package repo

import (
	"context"
	"testing"
	"time"
)

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
