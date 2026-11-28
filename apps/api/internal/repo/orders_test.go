package repo

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"greentravelpoints/internal/app"
)

// 集成测试：真连 MySQL，覆盖兑换 / 取消 / 并发防超卖 / 对账 / 过期清算。
//
// 运行方式（CI 已起 mysql:8.4 service 并注入 REPO_TEST_DSN；本地未设置时全部跳过）：
//
//	REPO_TEST_DSN='greentravel:greentravel@tcp(127.0.0.1:3306)/green_travel_points_test?parseTime=true&loc=Local&charset=utf8mb4' \
//	  go test ./internal/repo -count=1 -race
//
// 一次性准备（需要 MySQL 管理员）：
//
//	CREATE DATABASE IF NOT EXISTS green_travel_points_test
//	  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
//	GRANT ALL PRIVILEGES ON green_travel_points_test.* TO 'greentravel'@'localhost';
//	# 再按文件名顺序执行 db/migrations/*.up.sql
//
// 安全约束：库名必须以 _test 结尾——这些用例会整表清空，指错库会把演示数据洗掉。
const testDSNEnv = "REPO_TEST_DSN"

// 固定夹具 ID，与种子数据（1001–1003）互不重叠。
const (
	testUserID    = int64(9001)
	testAccountID = int64(9001)
	testMerchant  = int64(9001)
	testProductID = int64(9001)
	testSKUID     = int64(9001)
	testBatchOld  = int64(9001) // 先到期：400 分
	testBatchNew  = int64(9002) // 后到期：600 分
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(testDSNEnv))
	if dsn == "" {
		t.Skip("未设置 " + testDSNEnv + "，跳过 repo 集成测试（需要专用 MySQL 库，库名以 _test 结尾）")
	}
	parsedDSN, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", testDSNEnv, err)
	}
	if !strings.HasSuffix(parsedDSN.DBName, "_test") {
		t.Fatalf("%s 的数据库名必须以 _test 结尾，避免整表清空洗掉非测试数据", testDSNEnv)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	// 并发用例需要多个连接，同时别把池开太大。
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Ping(); err != nil {
		t.Fatalf("连接测试库失败（DSN 是否正确？）: %v", err)
	}

	store := New(db)
	resetFixture(t, db)
	return store
}

// resetFixture 清空业务表并写入固定夹具：
// 账户余额 1000 = 批次 400（先到期）+ 600（后到期），SKU 单价 300、库存 10、不限购。
func resetFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	// 无外键约束，清空顺序只按可读性排列；仅允许在 _test 库执行（见 newTestStore）。
	for _, stmt := range []string{
		`DELETE FROM audit_log`,
		`DELETE FROM point_batch_txn`,
		`DELETE FROM point_txn`,
		`DELETE FROM point_batch`,
		`DELETE FROM point_account`,
		`DELETE FROM redeem_order`,
		`DELETE FROM event_outbox`,
		`DELETE FROM carbon_record`,
		`DELETE FROM trip_record`,
		`DELETE FROM product_sku`,
		`DELETE FROM product`,
		`DELETE FROM merchant`,
		"DELETE FROM `user`",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("清空夹具失败 %q: %v", stmt, err)
		}
	}

	for _, stmt := range []string{
		`INSERT INTO user (id, phone, nickname, password_hash, role, status)
		 VALUES (9001, '13800009999', '集成测试用户', 'sha256$00$00', 'CITIZEN', 'ACTIVE')`,
		`INSERT INTO point_account (id, user_id, balance, frozen, total_saved_g, version)
		 VALUES (9001, 9001, 1000, 0, 0, 0)`,
		// 两笔历史发放流水，保证 Σ(txn) = 余额、且两批次各有独立来源流水（grant_txn_id 唯一）。
		`INSERT INTO point_txn (id, account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
		 VALUES (9002, 9001, 'TRAVEL', 400, 400, 'trip_record', 9002),
		        (9003, 9001, 'TRAVEL', 600, 1000, 'trip_record', 9003)`,
		`INSERT INTO point_batch (id, account_id, grant_txn_id, granted_amount, remaining, expires_at, status)
		 VALUES (9001, 9001, 9002, 400, 400, DATE_ADD(NOW(3), INTERVAL 10 DAY), 'ACTIVE'),
		        (9002, 9001, 9003, 600, 600, DATE_ADD(NOW(3), INTERVAL 300 DAY), 'ACTIVE')`,
		`INSERT INTO merchant (id, name, contact, settle_cycle, status)
		 VALUES (9001, '集成测试商户', '测试 13800009999', 'MONTHLY', 'ACTIVE')`,
		`INSERT INTO product (id, merchant_id, name, description, category, cover_url, status, sort_order, limit_per_user)
		 VALUES (9001, 9001, '集成测试商品', '仅测试用', '文创', '', 'ONLINE', 10, 0)`,
		`INSERT INTO product_sku (id, product_id, sku_name, points_price, stock, sold_count)
		 VALUES (9001, 9001, '默认款', 300, 10, 0)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("写入夹具失败 %q: %v", stmt, err)
		}
	}
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("执行 SQL 失败 %q: %v", query, err)
	}
}

func mustQueryInt(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("查询失败 %q: %v", query, err)
	}
	return n
}

func newOrderInput() app.CreateOrderInput {
	return app.CreateOrderInput{
		UserID:    testUserID,
		ProductID: testProductID,
		SKUId:     testSKUID,
		Receiver:  "集成测试",
	}
}

// assertBatchLedgerBalanced 校验批次恒等式：Σ(point_account.balance) == Σ(point_batch.remaining WHERE ACTIVE)。
func assertBatchLedgerBalanced(t *testing.T, db *sql.DB) {
	t.Helper()
	diff := mustQueryInt(t, db,
		`SELECT (SELECT COALESCE(SUM(balance), 0) FROM point_account)
		      - (SELECT COALESCE(SUM(remaining), 0) FROM point_batch WHERE status = 'ACTIVE')`)
	if diff != 0 {
		t.Fatalf("批次恒等式不成立：Σbalance − Σremaining(ACTIVE) = %d，期望 0", diff)
	}
}

// TestCreateOrderSuccess 兑换成功：扣分、扣库存、按 FIFO 消耗最早到期的批次并记明细。
func TestCreateOrderSuccess(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	order, replay, err := store.CreateOrder(ctx, newOrderInput())
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	if replay {
		t.Fatal("首次下单不应是幂等重放")
	}
	if order.Status != "PENDING" || order.PointsSpent != 300 || order.Quantity != 1 {
		t.Fatalf("订单字段不符: %+v", order)
	}

	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 700 {
		t.Fatalf("余额 = %d，期望 700", got)
	}
	// 300 分全部从先到期的批次 9001 扣（400 → 100），不动 9002。
	if got := mustQueryInt(t, store.db, `SELECT remaining FROM point_batch WHERE id = ?`, testBatchOld); got != 100 {
		t.Fatalf("早批次剩余 = %d，期望 100", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT remaining FROM point_batch WHERE id = ?`, testBatchNew); got != 600 {
		t.Fatalf("晚批次剩余 = %d，期望 600（不应被扣）", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch_txn WHERE batch_id = ? AND amount = -300`, testBatchOld); got != 1 {
		t.Fatalf("批次消耗明细条数 = %d，期望 1", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT stock FROM product_sku WHERE id = ?`, testSKUID); got != 9 {
		t.Fatalf("库存 = %d，期望 9", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM event_outbox WHERE topic = 'OrderPaid'`); got != 1 {
		t.Fatalf("OrderPaid 事件数 = %d，期望 1", got)
	}
	assertBatchLedgerBalanced(t, store.db)
}

// TestCreateOrderInsufficientPoints 积分不足：返回哨兵错误，不产生订单、不动余额与库存。
func TestCreateOrderInsufficientPoints(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE point_account SET balance = 100 WHERE id = ?`, testAccountID)
	mustExec(t, store.db, `UPDATE point_batch SET remaining = 100 WHERE id = ?`, testBatchOld)
	mustExec(t, store.db, `DELETE FROM point_batch WHERE id = ?`, testBatchNew)

	_, _, err := store.CreateOrder(ctx, newOrderInput())
	if !errors.Is(err, app.ErrInsufficientPoints) {
		t.Fatalf("期望 ErrInsufficientPoints，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM redeem_order`); got != 0 {
		t.Fatalf("订单数 = %d，期望 0", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT stock FROM product_sku WHERE id = ?`, testSKUID); got != 10 {
		t.Fatalf("库存 = %d，期望 10（不应扣减）", got)
	}
}

// TestCreateOrderRejectsInsufficientBatches 防止账实异常时只扣余额不扣批次。
func TestCreateOrderRejectsInsufficientBatches(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE point_batch SET remaining = 100 WHERE id = ?`, testBatchOld)
	mustExec(t, store.db, `DELETE FROM point_batch WHERE id = ?`, testBatchNew)

	_, _, err := store.CreateOrder(ctx, newOrderInput())
	if !errors.Is(err, app.ErrInsufficientPoints) {
		t.Fatalf("期望 ErrInsufficientPoints，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM redeem_order`); got != 0 {
		t.Fatalf("订单数 = %d，期望 0", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 1000 {
		t.Fatalf("余额 = %d，期望 1000（不应扣分）", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT stock FROM product_sku WHERE id = ?`, testSKUID); got != 10 {
		t.Fatalf("库存 = %d，期望 10（不应扣减）", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM point_txn WHERE txn_type = 'REDEEM'`); got != 0 {
		t.Fatalf("兑换流水数 = %d，期望 0", got)
	}
}

// TestCreateOrderOutOfStock 库存为 0：返回 OUT_OF_STOCK，不扣分。
func TestCreateOrderOutOfStock(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE product_sku SET stock = 0 WHERE id = ?`, testSKUID)

	_, _, err := store.CreateOrder(ctx, newOrderInput())
	if !errors.Is(err, app.ErrOutOfStock) {
		t.Fatalf("期望 ErrOutOfStock，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 1000 {
		t.Fatalf("余额 = %d，期望 1000（不应扣分）", got)
	}
}

// TestCreateOrderUnknownProductNotFound 商品不存在：返回 NOT_FOUND，而不是误报 OUT_OF_STOCK。
// 回归 DEF-02：productId 不存在时，按 product_id 查 SKU 无行会被当成“无库存”。
func TestCreateOrderUnknownProductNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	in := newOrderInput()
	in.ProductID = 999999
	in.SKUId = 0

	_, _, err := store.CreateOrder(ctx, in)
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM redeem_order`); got != 0 {
		t.Fatalf("订单数 = %d，期望 0", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 1000 {
		t.Fatalf("余额 = %d，期望 1000（不应扣分）", got)
	}
}

// TestCreateOrderLimitPerUser 限购：limit_per_user=1 时第二单被拒。
func TestCreateOrderLimitPerUser(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE product SET limit_per_user = 1 WHERE id = ?`, testProductID)

	if _, _, err := store.CreateOrder(ctx, newOrderInput()); err != nil {
		t.Fatalf("首单应成功: %v", err)
	}
	_, _, err := store.CreateOrder(ctx, newOrderInput())
	if !errors.Is(err, app.ErrLimitExceeded) {
		t.Fatalf("期望 ErrLimitExceeded，实际 %v", err)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM redeem_order`); got != 1 {
		t.Fatalf("订单数 = %d，期望 1", got)
	}
}

// TestCancelOrderRestoresOriginalBatch 取消退款：补 REFUND 流水、余额与库存复原、
// 原批次按 point_batch_txn 原路回填且 expires_at 不变。
func TestCancelOrderRestoresOriginalBatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var expiresBefore time.Time
	if err := store.db.QueryRow(
		`SELECT expires_at FROM point_batch WHERE id = ?`, testBatchOld).Scan(&expiresBefore); err != nil {
		t.Fatalf("读取批次到期时间失败: %v", err)
	}

	order, _, err := store.CreateOrder(ctx, newOrderInput())
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	cancelled, err := store.CancelOrder(ctx, testUserID, order.ID, "集成测试取消")
	if err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Fatalf("状态 = %s，期望 CANCELLED", cancelled.Status)
	}

	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 1000 {
		t.Fatalf("余额 = %d，期望复原为 1000", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT remaining FROM point_batch WHERE id = ?`, testBatchOld); got != 400 {
		t.Fatalf("批次剩余 = %d，期望回填为 400", got)
	}
	var expiresAfter time.Time
	if err := store.db.QueryRow(
		`SELECT expires_at FROM point_batch WHERE id = ?`, testBatchOld).Scan(&expiresAfter); err != nil {
		t.Fatalf("读取批次到期时间失败: %v", err)
	}
	if !expiresAfter.Equal(expiresBefore) {
		t.Fatalf("回填不应改变到期时间：before=%v after=%v", expiresBefore, expiresAfter)
	}
	if got := mustQueryInt(t, store.db, `SELECT stock FROM product_sku WHERE id = ?`, testSKUID); got != 10 {
		t.Fatalf("库存 = %d，期望回补为 10", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM point_txn WHERE txn_type = 'REFUND'`); got != 1 {
		t.Fatalf("REFUND 流水数 = %d，期望 1", got)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COUNT(*) FROM point_batch_txn WHERE batch_id = ? AND amount > 0`, testBatchOld); got != 1 {
		t.Fatalf("回填明细数 = %d，期望 1", got)
	}
	assertBatchLedgerBalanced(t, store.db)
}

// TestCreateOrderConcurrentNoOversell 20 并发抢 5 件库存：成功恰好 5 单，库存不为负、账本仍平。
func TestCreateOrderConcurrentNoOversell(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	mustExec(t, store.db, `UPDATE product_sku SET stock = 5, points_price = 100 WHERE id = ?`, testSKUID)

	const goroutines = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
		failures  []error
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _, err := store.CreateOrder(ctx, newOrderInput())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failures = append(failures, err)
				return
			}
			successes++
		}()
	}
	wg.Wait()

	if successes != 5 {
		t.Fatalf("成功单数 = %d，期望 5（库存 5）；失败 %d 次，首个错误 %v", successes, len(failures), firstErr(failures))
	}
	for _, err := range failures {
		if !errors.Is(err, app.ErrOutOfStock) {
			t.Fatalf("失败原因应全部是 ErrOutOfStock，实际 %v", err)
		}
	}
	if got := mustQueryInt(t, store.db, `SELECT stock FROM product_sku WHERE id = ?`, testSKUID); got != 0 {
		t.Fatalf("库存 = %d，期望 0（不允许超卖）", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT sold_count FROM product_sku WHERE id = ?`, testSKUID); got != 5 {
		t.Fatalf("销量 = %d，期望 5", got)
	}
	if got := mustQueryInt(t, store.db, `SELECT COUNT(*) FROM redeem_order`); got != 5 {
		t.Fatalf("订单数 = %d，期望 5", got)
	}
	// 5 × 100 分：余额 1000 → 500；批次同步减少，恒等式保持。
	if got := mustQueryInt(t, store.db, `SELECT balance FROM point_account WHERE id = ?`, testAccountID); got != 500 {
		t.Fatalf("余额 = %d，期望 500", got)
	}
	assertBatchLedgerBalanced(t, store.db)
}

func firstErr(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}

// TestReconcileAfterOrderAndCancel 对账：下单 + 取消后，账户↔流水、账户↔批次两个口径都平。
func TestReconcileAfterOrderAndCancel(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	order, _, err := store.CreateOrder(ctx, newOrderInput())
	if err != nil {
		t.Fatalf("下单失败: %v", err)
	}
	if res, err := store.ReconcilePoints(ctx); err != nil {
		t.Fatalf("对账失败: %v", err)
	} else if !res.OK || len(res.Mismatches) != 0 {
		t.Fatalf("下单后对账不一致: %+v", res)
	}
	assertBatchLedgerBalanced(t, store.db)

	if _, err := store.CancelOrder(ctx, testUserID, order.ID, ""); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	res, err := store.ReconcilePoints(ctx)
	if err != nil {
		t.Fatalf("对账失败: %v", err)
	}
	if !res.OK || len(res.Mismatches) != 0 {
		t.Fatalf("取消后对账不一致: %+v", res)
	}
	if got := mustQueryInt(t, store.db,
		`SELECT COALESCE(SUM(amount), 0) FROM point_txn`); got != 1000 {
		t.Fatalf("Σ流水 = %d，期望 1000（发放 1000 − 兑换 300 + 退款 300）", got)
	}
	assertBatchLedgerBalanced(t, store.db)
}
