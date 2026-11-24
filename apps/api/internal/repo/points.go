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
