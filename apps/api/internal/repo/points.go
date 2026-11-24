package repo

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"greentravelpoints/internal/app"
)

// txnTypeAliases 兼容前端历史枚举（EARN/SPEND/REVERSE）与 DB 枚举。
var txnTypeAliases = map[string]string{
	"EARN":    "TRAVEL",
	"SPEND":   "REDEEM",
	"REVERSE": "REFUND",
}

// txnReason 把流水类型翻译成前端可直接展示的说明。
var txnReason = map[string]string{
	"TRAVEL": "行程奖励",
	"REDEEM": "积分兑换",
	"REFUND": "兑换退款",
	"ADJUST": "人工调整",
	"EXPIRE": "积分过期",
}

// Summary 返回积分概览（M5）。
//
// 口径：
//   - balance / frozen / total_saved_g 取 point_account 快照；
//   - earnedThisMonth / earnedToday 统计 point_txn 中正数入账（含 TRAVEL / ADJUST）；
//   - dailyCap 取 sys_config.daily_points_cap，0 表示不限制。
func (s *Store) Summary(ctx context.Context, userID int64) (app.PointsSummary, error) {
	out := app.PointsSummary{DailyCap: s.dailyCap(ctx)}

	var balance, frozen int64
	var totalSaved float64
	err := s.db.QueryRowContext(ctx,
		`SELECT balance, frozen, total_saved_g FROM point_account WHERE user_id = ?`, userID,
	).Scan(&balance, &frozen, &totalSaved)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, nil // 新用户还没有账户，返回全 0
		}
		return out, err
	}
	out.Balance = balance
	out.Frozen = frozen
	out.Available = balance - frozen
	out.TotalCarbonSavedGrams = totalSaved

	err = s.db.QueryRowContext(ctx,
		`SELECT
		   COALESCE(SUM(CASE WHEN t.occurred_at >= DATE_FORMAT(CURDATE(), '%Y-%m-01') THEN t.amount END), 0),
		   COALESCE(SUM(CASE WHEN t.occurred_at >= CURDATE() THEN t.amount END), 0)
		 FROM point_txn t
		 JOIN point_account a ON a.id = t.account_id
		 WHERE a.user_id = ? AND t.amount > 0 AND t.txn_type <> 'REFUND'`, userID,
	).Scan(&out.EarnedThisMonth, &out.EarnedToday)
	if err != nil {
		return out, err
	}
	return out, nil
}

// ListTransactions 分页返回积分流水（M5）。
func (s *Store) ListTransactions(ctx context.Context, q app.TxnQuery) (app.Page[app.PointTxn], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	types := canonicalTxnTypes(q.Types)

	where := []string{"a.user_id = ?"}
	args := []any{q.UserID}
	if len(types) > 0 {
		where = append(where, "t.txn_type IN ("+placeholders(len(types))+")")
		for _, t := range types {
			args = append(args, t)
		}
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM point_txn t JOIN point_account a ON a.id = t.account_id WHERE `+whereSQL,
		args...,
	).Scan(&total); err != nil {
		return app.Page[app.PointTxn]{}, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.txn_type, t.amount, t.balance_after, t.biz_ref_type, t.biz_ref_id, t.occurred_at
		 FROM point_txn t
		 JOIN point_account a ON a.id = t.account_id
		 WHERE `+whereSQL+`
		 ORDER BY t.id DESC
		 LIMIT ? OFFSET ?`,
		append(args, pageSize, (page-1)*pageSize)...,
	)
	if err != nil {
		return app.Page[app.PointTxn]{}, err
	}
	defer rows.Close()

	items := make([]app.PointTxn, 0, pageSize)
	for rows.Next() {
		var it app.PointTxn
		if err := rows.Scan(&it.ID, &it.Type, &it.Amount, &it.BalanceAfter,
			&it.RefType, &it.RefID, &it.CreatedAt); err != nil {
			return app.Page[app.PointTxn]{}, err
		}
		it.Reason = txnReason[it.Type]
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.PointTxn]{}, err
	}
	return app.Page[app.PointTxn]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// dailyCap 读取单日积分上限，读不到或非法时返回 0（不限制）。
func (s *Store) dailyCap(ctx context.Context) int64 {
	var raw string
	if err := s.db.QueryRowContext(ctx,
		`SELECT config_value FROM sys_config WHERE config_key = 'daily_points_cap'`,
	).Scan(&raw); err != nil {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// DailyCarbon 返回最近 days 天逐日碳减排量（缺失日期补 0），供前端趋势图使用。
func (s *Store) DailyCarbon(ctx context.Context, userID int64, days int) ([]app.DailyCarbonPoint, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -(days - 1))

	rows, err := s.db.QueryContext(ctx,
		`SELECT DATE_FORMAT(cr.computed_at, '%Y-%m-%d') AS d,
		        COALESCE(SUM(cr.carbon_saved_g), 0) AS grams
		 FROM carbon_record cr
		 JOIN trip_record tr ON tr.id = cr.trip_id
		 WHERE tr.user_id = ? AND tr.status = 'VALID' AND cr.computed_at >= ?
		 GROUP BY d`, userID, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDate := make(map[string]float64, days)
	for rows.Next() {
		var date string
		var grams float64
		if err := rows.Scan(&date, &grams); err != nil {
			return nil, err
		}
		byDate[date] = grams
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]app.DailyCarbonPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, app.DailyCarbonPoint{Date: d, Grams: byDate[d]})
	}
	return out, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// canonicalTxnTypes 去掉未知值、把别名映射为 DB 枚举并去重。
func canonicalTxnTypes(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t := strings.ToUpper(strings.TrimSpace(raw))
		if alias, ok := txnTypeAliases[t]; ok {
			t = alias
		}
		switch t {
		case "TRAVEL", "REDEEM", "REFUND", "ADJUST", "EXPIRE":
		default:
			continue
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// ExpirePoints 执行积分过期清算：把已到期且仍有余量的 ACTIVE 批次逐条置为 EXPIRED，
// 同时写 EXPIRE 反向流水、扣减账户余额，并在 point_batch_txn 记一笔批次扣减明细。
//
// 语义与不变量：
//   - 每批次最多一条 EXPIRE 流水：biz_ref_type='point_batch' + biz_ref_id=batch_id，
//     由 uk_point_txn_biz 兜底，重复执行不会重复扣分；
//   - 过期量取 min(batch.remaining, account.balance)：历史数据账实不符（余额小于批次余量）时
//     只扣得动的部分，批次仍置 EXPIRED 并归零，差额留给 /admin/points/reconcile 暴露；
//   - 全程单事务，锁顺序 account → batch（与 CreateOrder 一致，避免交叉死锁）；
//   - 结束后 Σpoint_account.balance 与 Σpoint_batch.remaining(ACTIVE) 仍然相等（正常数据下）。
func (s *Store) ExpirePoints(ctx context.Context) (app.ExpireResult, error) {
	out := app.ExpireResult{}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()

	// 先固定账户顺序（升序）再逐个加锁，避免与下单/取消事务交叉成环。
	accountIDs, err := expiredBatchAccounts(ctx, tx)
	if err != nil {
		return out, err
	}

	now := time.Now()
	for _, accountID := range accountIDs {
		var balance int64
		err := tx.QueryRowContext(ctx,
			`SELECT balance FROM point_account WHERE id = ? FOR UPDATE`, accountID).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			continue // 批次指向的账户已不存在（历史脏数据），跳过
		}
		if err != nil {
			return out, err
		}

		batches, err := lockExpiredBatches(ctx, tx, accountID)
		if err != nil {
			return out, err
		}
		balanceChanged := false
		for _, b := range batches {
			expire := b.remaining
			if expire > balance {
				expire = balance
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE point_batch SET remaining = 0, status = 'EXPIRED' WHERE id = ?`, b.id); err != nil {
				return out, err
			}
			out.Batches++
			if expire <= 0 {
				continue // 余额已为 0：批次核销掉，但不再写金额为 0 的流水（DB 约束 amount <> 0）
			}
			balance -= expire
			balanceChanged = true
			res, err := tx.ExecContext(ctx,
				`INSERT INTO point_txn
				   (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id, occurred_at)
				 VALUES (?, 'EXPIRE', ?, ?, 'point_batch', ?, ?)`,
				accountID, -expire, balance, b.id, now)
			if err != nil {
				return out, err
			}
			txnID, err := res.LastInsertId()
			if err != nil {
				return out, err
			}
			// 明细同样记账：取消订单回填时按 point_batch_txn 原路还原，不能把已过期的量算回去。
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO point_batch_txn (batch_id, txn_id, amount) VALUES (?, ?, ?)`,
				b.id, txnID, -expire); err != nil {
				return out, err
			}
			out.Points += expire
		}
		if balanceChanged {
			if _, err := tx.ExecContext(ctx,
				`UPDATE point_account SET balance = ?, version = version + 1 WHERE id = ?`,
				balance, accountID); err != nil {
				return out, err
			}
			out.Accounts++
		}
	}

	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}

// expiredBatchAccounts 列出存在到期批次的账户，按 id 升序固定加锁顺序。
func expiredBatchAccounts(ctx context.Context, tx *sql.Tx) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT account_id FROM point_batch
		 WHERE status = 'ACTIVE' AND remaining > 0 AND expires_at <= NOW(3)
		 ORDER BY account_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0, 16)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// expiredBatch 是一条待过期批次的快照（id + 剩余量）。
type expiredBatch struct{ id, remaining int64 }

// lockExpiredBatches 锁住某账户下所有已到期批次，按 expires_at、id 升序（与 FIFO 口径一致）。
func lockExpiredBatches(ctx context.Context, tx *sql.Tx, accountID int64) ([]expiredBatch, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, remaining FROM point_batch
		 WHERE account_id = ? AND status = 'ACTIVE' AND remaining > 0 AND expires_at <= NOW(3)
		 ORDER BY expires_at ASC, id ASC FOR UPDATE`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]expiredBatch, 0, 8)
	for rows.Next() {
		var b expiredBatch
		if err := rows.Scan(&b.id, &b.remaining); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}
