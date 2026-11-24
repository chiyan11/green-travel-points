package repo

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"greentravelpoints/internal/app"
)

// 人工调账与审计的仓库级用例（BR-14 / NFR-12）：需要真库，未设置 REPO_TEST_DSN 时跳过，
// 夹具与准备步骤见 orders_test.go 顶部注释。

// TestAdjustPointsWritesAudit 调增：余额 / 批次 / 流水 / 审计四处一致，且可按操作人与动作反查。
func TestAdjustPointsWritesAudit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	const operatorID = int64(1002)
	const reason = "客服补偿：行程漏记"
	res, err := store.AdjustPoints(ctx, operatorID, app.AdjustPointsInput{
		UserID: testUserID,
		Amount: 200,
		Reason: reason,
	})
	if err != nil {
		t.Fatalf("调账失败: %v", err)
	}
	if res.Balance != 1200 || res.UserID != testUserID || res.AuditID <= 0 {
		t.Fatalf("调账结果不符: %+v", res)
	}

	// 流水：ADJUST +200，回指审计记录，余额快照一致。
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_txn
		 WHERE txn_type = 'ADJUST' AND amount = 200 AND balance_after = 1200
		   AND biz_ref_type = 'audit_log' AND biz_ref_id = ?`, res.AuditID); got != 1 {
		t.Fatalf("ADJUST 流水不符：期望 1 条回指 audit_log=%d 的 +200 流水", res.AuditID)
	}

	// 批次：调增必须同时写批次，否则批次恒等式会破。
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch
		 WHERE account_id = ? AND remaining = 200 AND status = 'ACTIVE'`, testAccountID); got != 1 {
		t.Fatalf("调增未写批次：期望一条 remaining=200 的 ACTIVE 批次")
	}
	assertBatchLedgerBalanced(t, store.db)

	// 审计：操作人 / 前后值 / 金额 / 理由齐全（BR-14）。
	var (
		gotOperator, gotAmount         int64
		gotBefore, gotAfter, gotReason string
		gotTargetKey, gotAction        string
	)
	err = store.db.QueryRow(`SELECT operator_id, action, target_key, amount, before_value, after_value, reason
	                         FROM audit_log WHERE id = ?`, res.AuditID).
		Scan(&gotOperator, &gotAction, &gotTargetKey, &gotAmount, &gotBefore, &gotAfter, &gotReason)
	if err != nil {
		t.Fatalf("读取审计记录失败: %v", err)
	}
	if gotOperator != operatorID || gotAmount != 200 || gotBefore != "1000" || gotAfter != "1200" ||
		gotReason != reason || gotAction != "ADJUST" ||
		gotTargetKey != strconv.FormatInt(testUserID, 10) {
		t.Fatalf("审计字段不符: operator=%d action=%s key=%s amount=%d before=%s after=%s reason=%q",
			gotOperator, gotAction, gotTargetKey, gotAmount, gotBefore, gotAfter, gotReason)
	}

	// 审计查询：按操作人 + 动作（小写也能命中）反查得到这一条。
	page, err := store.ListAuditLogs(ctx, app.AuditQuery{OperatorID: operatorID, Action: "adjust"})
	if err != nil {
		t.Fatalf("审计查询失败: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != res.AuditID ||
		page.Items[0].Reason != reason {
		t.Fatalf("审计查询结果不符: %+v", page)
	}

	// 账户↔流水对账仍平。
	if rec, err := store.ReconcilePoints(ctx); err != nil || !rec.OK {
		t.Fatalf("调账后对账不一致: %+v, err=%v", rec, err)
	}
}

// TestAdjustPointsInsufficient 扣减超过余额：返回哨兵错误，且不落审计、不改余额。
func TestAdjustPointsInsufficient(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.AdjustPoints(ctx, 1002, app.AdjustPointsInput{
		UserID: testUserID,
		Amount: -5000,
		Reason: "超额扣减",
	})
	if !errors.Is(err, app.ErrInsufficientPoints) {
		t.Fatalf("期望 ErrInsufficientPoints，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM audit_log`); got != 0 {
		t.Fatalf("失败调账不应留审计，实际 %d 条", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 1000 {
		t.Fatalf("余额 = %d，期望仍为 1000", got)
	}
}

// TestAdjustPointsReducesBatchesFIFO 调减：按 FIFO 消耗最早到期批次，并写 point_batch_txn 明细。
func TestAdjustPointsReducesBatchesFIFO(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	res, err := store.AdjustPoints(ctx, 1002, app.AdjustPointsInput{
		UserID: testUserID,
		Amount: -300,
		Reason: "刷分扣回",
	})
	if err != nil {
		t.Fatalf("调减失败: %v", err)
	}
	if res.Balance != 700 {
		t.Fatalf("余额 = %d，期望 700", res.Balance)
	}
	// 早批次 400 → 100，晚批次 600 不动。
	if got := mustQueryInt(t, store.db, `SELECT remaining FROM point_batch WHERE id = ?`, testBatchOld); got != 100 {
		t.Fatalf("早批次剩余 = %d，期望 100", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT remaining FROM point_batch WHERE id = ?`, testBatchNew); got != 600 {
		t.Fatalf("晚批次剩余 = %d，期望 600（不应被扣）", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch_txn WHERE batch_id = ? AND amount = -300`, testBatchOld); got != 1 {
		t.Fatalf("批次消耗明细缺失：期望 1 条 -300")
	}
	assertBatchLedgerBalanced(t, store.db)
}
