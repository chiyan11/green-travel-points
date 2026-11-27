package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"greentravelpoints/internal/app"
)

// P1 回归：幂等键按用户隔离、同值更新不误判 404、字段超长/重名转业务错误、
// 过期清算不静默抹差、对账同时核批次余额与批次流水。

// TestCreateOrderSameIdempotencyKeyAcrossUsers 同一幂等键在不同用户下应各自成单。
func TestCreateOrderSameIdempotencyKeyAcrossUsers(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	const otherUserID = int64(9002)
	mustExec(t, store.db, `INSERT INTO user (id, phone, nickname, password_hash, role, status)
		 VALUES (?, '13800009998', '第二个测试用户', 'sha256$00$00', 'CITIZEN', 'ACTIVE')`, otherUserID)
	mustExec(t, store.db, `INSERT INTO point_account (id, user_id, balance, frozen, total_saved_g, version)
		 VALUES (?, ?, 1000, 0, 0, 0)`, otherUserID, otherUserID)
	mustExec(t, store.db, `INSERT INTO point_txn (id, account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
		 VALUES (9101, ?, 'TRAVEL', 1000, 1000, 'trip_record', 9101)`, otherUserID)
	mustExec(t, store.db, `INSERT INTO point_batch (id, account_id, grant_txn_id, granted_amount, remaining, expires_at, status)
		 VALUES (9101, ?, 9101, 1000, 1000, DATE_ADD(NOW(3), INTERVAL 30 DAY), 'ACTIVE')`, otherUserID)

	const key = "same-key-across-users"
	first, _, err := store.CreateOrder(ctx, app.CreateOrderInput{
		UserID:         testUserID,
		ProductID:      testProductID,
		SKUId:          testSKUID,
		Receiver:       "用户一",
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("用户一下单失败: %v", err)
	}
	second, _, err := store.CreateOrder(ctx, app.CreateOrderInput{
		UserID:         otherUserID,
		ProductID:      testProductID,
		SKUId:          testSKUID,
		Receiver:       "用户二",
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("用户二复用同一幂等键不应失败: %v", err)
	}
	if first.UserID != testUserID || second.UserID != otherUserID {
		t.Fatalf("订单归属不符: first=%d second=%d", first.UserID, second.UserID)
	}
	if first.ID == second.ID {
		t.Fatalf("同一幂等键在不同用户下生成了同一订单: %d", first.ID)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM redeem_order WHERE idempotency_key = ?`, key); got != 2 {
		t.Fatalf("同一幂等键订单数 = %d，期望 2", got)
	}
}

// TestAdminSameValueUpdateIsNotNotFound 同值更新 RowsAffected=0 不能当成 404。
func TestAdminSameValueUpdateIsNotNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	sameName := "集成测试商品"
	product, err := store.UpdateProduct(ctx, testProductID, app.ProductUpdate{Name: &sameName})
	if err != nil {
		t.Fatalf("同值更新商品不应 404: %v", err)
	}
	if product.ID != testProductID || product.Name != sameName {
		t.Fatalf("同值更新返回商品不符: %+v", product)
	}

	sku, err := store.SetSKUStock(ctx, testSKUID, 10)
	if err != nil {
		t.Fatalf("同值更新库存不应 404: %v", err)
	}
	if sku.ID != testSKUID || sku.Stock != 10 {
		t.Fatalf("同值更新库存返回不符: %+v", sku)
	}
}

// TestAdminRepoTranslatesDataErrors 超长字段与重名必须转成业务错误，而不是 500。
func TestAdminRepoTranslatesDataErrors(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	validSKU := app.AdminSKUInput{Name: "默认", PointsPrice: 100, Stock: 1}

	_, err := store.CreateProduct(ctx, app.AdminProductInput{
		MerchantID: testMerchant,
		Name:       strings.Repeat("商", 129),
		Category:   "文创",
		SKUs:       []app.AdminSKUInput{validSKU},
	})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("超长商品名期望 ErrInvalidInput，实际 %v", err)
	}

	_, err = store.CreateProduct(ctx, app.AdminProductInput{
		MerchantID: testMerchant,
		Name:       "集成测试商品",
		Category:   "文创",
		SKUs:       []app.AdminSKUInput{validSKU},
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("同商户重名期望 ErrConflict，实际 %v", err)
	}

	longDesc := strings.Repeat("d", 501)
	_, err = store.UpdateProduct(ctx, testProductID, app.ProductUpdate{Description: &longDesc})
	if !errors.Is(err, app.ErrInvalidInput) {
		t.Fatalf("超长描述期望 ErrInvalidInput，实际 %v", err)
	}

	other, err := store.CreateProduct(ctx, app.AdminProductInput{
		MerchantID: testMerchant,
		Name:       "第二个商品",
		Category:   "文创",
		SKUs:       []app.AdminSKUInput{{Name: "默认", PointsPrice: 100, Stock: 1}},
	})
	if err != nil {
		t.Fatalf("准备第二个商品失败: %v", err)
	}
	dupName := other.Name
	_, err = store.UpdateProduct(ctx, testProductID, app.ProductUpdate{Name: &dupName})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("改名撞同商户商品期望 ErrConflict，实际 %v", err)
	}
}

// TestExpirePointsRejectsInconsistentAccount 余额小于到期批次余量时整笔回滚，不静默抹差。
func TestExpirePointsRejectsInconsistentAccount(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db,
		`UPDATE point_batch SET expires_at = DATE_SUB(NOW(3), INTERVAL 1 DAY)`)
	mustExec(t, store.db, `UPDATE point_account SET balance = 500 WHERE id = ?`, testAccountID)

	if _, err := store.ExpirePoints(ctx); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("余额不足时期望 ErrConflict，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_txn WHERE txn_type = 'EXPIRE'`); got != 0 {
		t.Fatalf("失败清算不应写 EXPIRE 流水，实际 %d 条", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch WHERE status = 'ACTIVE' AND remaining > 0`); got != 2 {
		t.Fatalf("失败清算不应改批次，ACTIVE 批次数 = %d", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 500 {
		t.Fatalf("失败清算不应改余额，实际 %d", got)
	}
}

// TestExpirePointsFullBatchKeepsLedgerBalanced 整批过期后余额、批次、流水仍守恒。
func TestExpirePointsFullBatchKeepsLedgerBalanced(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db,
		`UPDATE point_batch SET expires_at = DATE_SUB(NOW(3), INTERVAL 1 DAY) WHERE id = ?`, testBatchOld)
	res, err := store.ExpirePoints(ctx)
	if err != nil {
		t.Fatalf("整批清算失败: %v", err)
	}
	if res.Batches != 1 || res.Points != 400 || res.Accounts != 1 {
		t.Fatalf("清算结果不符: %+v", res)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 600 {
		t.Fatalf("清算后余额 = %d，期望 600", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch WHERE id = ? AND status = 'EXPIRED' AND remaining = 0`, testBatchOld); got != 1 {
		t.Fatalf("早批次未正确过期")
	}
	assertBatchLedgerBalanced(t, store.db)
	if rec, err := store.ReconcilePoints(ctx); err != nil || !rec.OK {
		t.Fatalf("整批清算后对账不一致: %+v, err=%v", rec, err)
	}
}

// TestReconcileDetectsBatchMismatch 对账必须同时发现“余额与批次不等”和“批次与批次流水不等”。
func TestReconcileDetectsBatchMismatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE point_batch SET remaining = 399 WHERE id = ?`, testBatchOld)
	res, err := store.ReconcilePoints(ctx)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if res.OK || len(res.Mismatches) != 1 {
		t.Fatalf("期望 1 条不一致，实际 ok=%v mismatches=%+v", res.OK, res.Mismatches)
	}
	it := res.Mismatches[0]
	if it.AccountID != testAccountID || it.Balance != 1000 || it.BatchRemaining != 999 || it.BatchLedgerSum != 1000 {
		t.Fatalf("对账字段不符: %+v", it)
	}
	if !strings.Contains(it.Reason, "BATCH_BALANCE_MISMATCH") ||
		!strings.Contains(it.Reason, "BATCH_LEDGER_MISMATCH") {
		t.Fatalf("对账原因不符: %s", it.Reason)
	}
}

// TestReconcileDetectsBatchLedgerMismatch 余额与批次相等、但批次流水被篡改时也要报错。
func TestReconcileDetectsBatchLedgerMismatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db,
		`INSERT INTO point_batch_txn (batch_id, txn_id, amount) VALUES (?, 9002, 1)`, testBatchOld)
	res, err := store.ReconcilePoints(ctx)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if res.OK || len(res.Mismatches) != 1 {
		t.Fatalf("期望 1 条不一致，实际 ok=%v mismatches=%+v", res.OK, res.Mismatches)
	}
	it := res.Mismatches[0]
	if it.BatchRemaining != 1000 || it.BatchLedgerSum != 1001 {
		t.Fatalf("对账字段不符: %+v", it)
	}
	if !strings.Contains(it.Reason, "BATCH_LEDGER_MISMATCH") ||
		strings.Contains(it.Reason, "BATCH_BALANCE_MISMATCH") {
		t.Fatalf("对账原因不符: %s", it.Reason)
	}
}
