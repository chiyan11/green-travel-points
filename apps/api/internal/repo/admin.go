package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"greentravelpoints/internal/app"
	"greentravelpoints/internal/domain"
	"greentravelpoints/internal/eventbus"
)

// RoleOf 返回用户角色，供后台鉴权使用；用户不存在或已冻结返回 ErrNotFound。
func (s *Store) RoleOf(ctx context.Context, userID int64) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx,
		`SELECT role FROM user WHERE id = ? AND status = 'ACTIVE'`, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", app.ErrNotFound
	}
	return role, err
}

// ListAllProducts 后台商品列表：包含 DRAFT / OFFLINE，可按状态与关键词筛选（M8）。
func (s *Store) ListAllProducts(ctx context.Context, q app.AdminProductQuery) (app.Page[app.Product], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)

	where := []string{"1 = 1"}
	args := make([]any, 0, 3)
	if q.Status != "" {
		where = append(where, "p.status = ?")
		args = append(args, strings.ToUpper(q.Status))
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		where = append(where, "(p.name LIKE ? OR p.description LIKE ?)")
		args = append(args, like, like)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM product p WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return app.Page[app.Product]{}, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT p.id, p.merchant_id, p.name, p.description, p.category, p.cover_url, p.status,
		        p.limit_per_user, COALESCE(MIN(s.points_price), 0), COALESCE(SUM(s.stock), 0)
		 FROM product p LEFT JOIN product_sku s ON s.product_id = p.id
		 WHERE `+whereSQL+`
		 GROUP BY p.id, p.merchant_id, p.name, p.description, p.category, p.cover_url,
		          p.status, p.limit_per_user, p.sort_order
		 ORDER BY p.id DESC
		 LIMIT ? OFFSET ?`,
		append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return app.Page[app.Product]{}, err
	}
	defer rows.Close()

	items := make([]app.Product, 0, pageSize)
	for rows.Next() {
		var p app.Product
		if err := rows.Scan(&p.ID, &p.MerchantID, &p.Name, &p.Description, &p.Category,
			&p.ImageURL, &p.Status, &p.LimitPerUser, &p.PointsPrice, &p.Stock); err != nil {
			return app.Page[app.Product]{}, err
		}
		p.Active = p.Status == "ONLINE"
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.Product]{}, err
	}
	// 列表原先只给聚合价 / 总库存；后台「改库存」需要 skuId，这里批量补 SKU 明细。
	if err := s.attachSKUs(ctx, items); err != nil {
		return app.Page[app.Product]{}, err
	}
	return app.Page[app.Product]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// attachSKUs 为商品列表批量补 SKU 明细：一次 IN 查询，避免逐条 N+1。
func (s *Store) attachSKUs(ctx context.Context, items []app.Product) error {
	if len(items) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(items)), ",")
	args := make([]any, 0, len(items))
	index := make(map[int64]int, len(items))
	for i := range items {
		args = append(args, items[i].ID)
		index[items[i].ID] = i
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT product_id, id, sku_name, points_price, stock, sold_count
		 FROM product_sku WHERE product_id IN (`+placeholders+`)
		 ORDER BY product_id, id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var productID int64
		var sku app.SKU
		if err := rows.Scan(&productID, &sku.ID, &sku.Name, &sku.PointsPrice, &sku.Stock, &sku.SoldCount); err != nil {
			return err
		}
		if i, ok := index[productID]; ok {
			items[i].SKUs = append(items[i].SKUs, sku)
		}
	}
	return rows.Err()
}

// CreateProduct 后台新建商品（含 SKU），返回完整商品。
func (s *Store) CreateProduct(ctx context.Context, in app.AdminProductInput) (app.Product, error) {
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = "DRAFT"
	}
	if !validProductStatus(status) {
		return app.Product{}, app.ErrInvalidInput
	}
	var merchantCount int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM merchant WHERE id = ?`, in.MerchantID).Scan(&merchantCount); err != nil {
		return app.Product{}, err
	}
	if merchantCount == 0 {
		return app.Product{}, app.ErrNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.Product{}, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO product (merchant_id, name, description, category, cover_url, status, sort_order, limit_per_user)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.MerchantID, in.Name, in.Description, in.Category, in.ImageURL, status, in.SortOrder, in.LimitPerUser)
	if err != nil {
		if isDuplicate(err) {
			return app.Product{}, app.ErrConflict
		}
		return app.Product{}, err
	}
	productID, err := res.LastInsertId()
	if err != nil {
		return app.Product{}, err
	}
	for _, sku := range in.SKUs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO product_sku (product_id, sku_name, points_price, stock)
			 VALUES (?, ?, ?, ?)`, productID, sku.Name, sku.PointsPrice, sku.Stock); err != nil {
			if isDuplicate(err) {
				return app.Product{}, app.ErrConflict
			}
			return app.Product{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.Product{}, err
	}
	return s.getProductAny(ctx, productID)
}

// UpdateProduct 后台更新商品字段（nil 字段不更新）。
func (s *Store) UpdateProduct(ctx context.Context, id int64, in app.ProductUpdate) (app.Product, error) {
	sets := make([]string, 0, 8)
	args := make([]any, 0, 9)
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return app.Product{}, app.ErrInvalidInput
		}
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *in.Description)
	}
	if in.Category != nil {
		sets = append(sets, "category = ?")
		args = append(args, *in.Category)
	}
	if in.ImageURL != nil {
		sets = append(sets, "cover_url = ?")
		args = append(args, *in.ImageURL)
	}
	if in.Status != nil {
		st := strings.ToUpper(strings.TrimSpace(*in.Status))
		if !validProductStatus(st) {
			return app.Product{}, app.ErrInvalidInput
		}
		sets = append(sets, "status = ?")
		args = append(args, st)
	}
	if in.SortOrder != nil {
		sets = append(sets, "sort_order = ?")
		args = append(args, *in.SortOrder)
	}
	if in.LimitPerUser != nil {
		if *in.LimitPerUser < 0 {
			return app.Product{}, app.ErrInvalidInput
		}
		sets = append(sets, "limit_per_user = ?")
		args = append(args, *in.LimitPerUser)
	}
	if len(sets) == 0 {
		return s.getProductAny(ctx, id)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE product SET `+strings.Join(sets, ", ")+` WHERE id = ?`, append(args, id)...)
	if err != nil {
		return app.Product{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return app.Product{}, err
	}
	if affected == 0 {
		return app.Product{}, app.ErrNotFound
	}
	return s.getProductAny(ctx, id)
}

// SetSKUStock 后台设置 SKU 库存（绝对值）。
func (s *Store) SetSKUStock(ctx context.Context, skuID int64, stock int) (app.SKU, error) {
	if stock < 0 {
		return app.SKU{}, app.ErrInvalidInput
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE product_sku SET stock = ? WHERE id = ?`, stock, skuID)
	if err != nil {
		return app.SKU{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return app.SKU{}, err
	}
	if affected == 0 {
		return app.SKU{}, app.ErrNotFound
	}
	var sku app.SKU
	err = s.db.QueryRowContext(ctx,
		`SELECT id, sku_name, points_price, stock, sold_count FROM product_sku WHERE id = ?`, skuID,
	).Scan(&sku.ID, &sku.Name, &sku.PointsPrice, &sku.Stock, &sku.SoldCount)
	if err != nil {
		return app.SKU{}, err
	}
	return sku, nil
}

// ListAllOrders 后台订单列表（全部用户）。
func (s *Store) ListAllOrders(ctx context.Context, q app.AdminOrderQuery) (app.Page[app.Order], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)

	where := []string{"1 = 1"}
	args := make([]any, 0, 2)
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, strings.ToUpper(q.Status))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM redeem_order WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return app.Page[app.Order]{}, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+orderColumns+` FROM redeem_order WHERE `+whereSQL+`
		 ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return app.Page[app.Order]{}, err
	}
	defer rows.Close()

	items := make([]app.Order, 0, pageSize)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return app.Page[app.Order]{}, err
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.Order]{}, err
	}
	return app.Page[app.Order]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// TransitionOrder 按状态机执行后台流转（PENDING→PAID→SHIPPED→COMPLETED）。
func (s *Store) TransitionOrder(ctx context.Context, orderID int64, to, reason string) (app.Order, error) {
	target := domain.OrderStatus(strings.ToUpper(strings.TrimSpace(to)))
	if target != domain.OrderPaid && target != domain.OrderShipped && target != domain.OrderCompleted {
		return app.Order{}, app.ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.Order{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var current, orderNo string
	if err := tx.QueryRowContext(ctx,
		`SELECT status, order_no FROM redeem_order WHERE id = ? FOR UPDATE`, orderID,
	).Scan(&current, &orderNo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, app.ErrNotFound
		}
		return app.Order{}, err
	}
	if err := domain.CheckOrderTransition(domain.OrderStatus(strings.ToUpper(current)), target); err != nil {
		return app.Order{}, app.ErrIllegalTransition
	}

	switch target {
	case domain.OrderShipped:
		_, err = tx.ExecContext(ctx,
			`UPDATE redeem_order SET status = ?, shipped_at = NOW(3) WHERE id = ?`, string(target), orderID)
	case domain.OrderCompleted:
		_, err = tx.ExecContext(ctx,
			`UPDATE redeem_order SET status = ?, completed_at = NOW(3) WHERE id = ?`, string(target), orderID)
	default:
		_, err = tx.ExecContext(ctx,
			`UPDATE redeem_order SET status = ? WHERE id = ?`, string(target), orderID)
	}
	if err != nil {
		return app.Order{}, err
	}

	topic := map[domain.OrderStatus]string{
		domain.OrderPaid:      eventbus.TopicOrderConfirmed,
		domain.OrderShipped:   eventbus.TopicOrderShipped,
		domain.OrderCompleted: eventbus.TopicOrderCompleted,
	}[target]
	if err := insertOutbox(ctx, tx, topic, "order", orderNo, map[string]any{
		"orderNo": orderNo,
		"status":  string(target),
		"reason":  reason,
	}); err != nil {
		return app.Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return app.Order{}, err
	}
	return s.getOrder(ctx, orderID)
}

// ReconcilePoints 全量积分对账（BR-07）：返回不一致账户，作为报告"测试证据"。
func (s *Store) ReconcilePoints(ctx context.Context) (app.ReconcileResult, error) {
	out := app.ReconcileResult{Mismatches: make([]app.ReconcileItem, 0)}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM point_account`).Scan(&out.Accounts); err != nil {
		return app.ReconcileResult{}, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.user_id, a.balance, COALESCE(SUM(t.amount), 0) AS ledger_sum
		 FROM point_account a LEFT JOIN point_txn t ON t.account_id = a.id
		 GROUP BY a.id, a.user_id, a.balance
		 HAVING a.balance <> COALESCE(SUM(t.amount), 0)`)
	if err != nil {
		return app.ReconcileResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it app.ReconcileItem
		if err := rows.Scan(&it.AccountID, &it.UserID, &it.Balance, &it.LedgerSum); err != nil {
			return app.ReconcileResult{}, err
		}
		out.Mismatches = append(out.Mismatches, it)
	}
	if err := rows.Err(); err != nil {
		return app.ReconcileResult{}, err
	}
	out.OK = len(out.Mismatches) == 0
	return out, nil
}

// AdjustPoints 人工调增 / 调减积分（UC-15）。BR-14：必须填理由并写 audit_log，
// 记录操作人、账户、前后余额与理由；调账流水用 biz_ref_type='audit_log' 回指审计记录。
//
// 账本一致性（与批次恒等式对齐）：
//   - 调增：同时写 point_batch（有效期取 sys_config.points_expire_days），保证 Σbalance == Σremaining(ACTIVE)；
//   - 调减：按 FIFO 消耗批次；批次不足的部分只扣余额（历史数据常见），差额由对账接口暴露。
func (s *Store) AdjustPoints(ctx context.Context, operatorID int64, in app.AdjustPointsInput) (app.AdjustResult, error) {
	reason := strings.TrimSpace(in.Reason)
	if in.UserID <= 0 || in.Amount == 0 || reason == "" {
		return app.AdjustResult{}, fmt.Errorf("%w: userId / amount（非 0）/ reason 均为必填", app.ErrInvalidInput)
	}
	if operatorID <= 0 {
		return app.AdjustResult{}, fmt.Errorf("%w: 缺少操作人", app.ErrInvalidInput)
	}
	if len([]rune(reason)) > 255 {
		return app.AdjustResult{}, fmt.Errorf("%w: reason 超过 255 字", app.ErrInvalidInput)
	}

	// 账户不存在时先建空账户（事务外，避免并发下共享锁升级成死锁，同 CreateOrder）。
	if _, err := s.db.ExecContext(ctx,
		`INSERT IGNORE INTO point_account (user_id) VALUES (?)`, in.UserID); err != nil {
		return app.AdjustResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.AdjustResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var accountID, balance int64
	err = tx.QueryRowContext(ctx,
		`SELECT id, balance FROM point_account WHERE user_id = ? FOR UPDATE`, in.UserID,
	).Scan(&accountID, &balance)
	if errors.Is(err, sql.ErrNoRows) {
		return app.AdjustResult{}, app.ErrNotFound
	}
	if err != nil {
		return app.AdjustResult{}, err
	}

	after := balance + in.Amount
	if after < 0 {
		return app.AdjustResult{}, fmt.Errorf("%w: 余额 %d 不足以扣减 %d",
			app.ErrInsufficientPoints, balance, -in.Amount)
	}

	auditID, err := writeAudit(ctx, tx, auditEntry{
		operatorID:  operatorID,
		action:      "ADJUST",
		targetType:  "point_account",
		targetID:    accountID,
		targetKey:   strconv.FormatInt(in.UserID, 10), // 按用户检索审计
		amount:      in.Amount,
		beforeValue: strconv.FormatInt(balance, 10),
		afterValue:  strconv.FormatInt(after, 10),
		reason:      reason,
	})
	if err != nil {
		return app.AdjustResult{}, err
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO point_txn
		   (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
		 VALUES (?, 'ADJUST', ?, ?, 'audit_log', ?)`,
		accountID, in.Amount, after, auditID)
	if err != nil {
		return app.AdjustResult{}, err
	}
	txnID, err := res.LastInsertId()
	if err != nil {
		return app.AdjustResult{}, err
	}

	if in.Amount > 0 {
		days := s.pointsExpireDays(ctx)
		if err := GrantPointsWithBatch(ctx, tx, accountID, txnID, in.Amount,
			time.Now().AddDate(0, 0, int(days))); err != nil {
			return app.AdjustResult{}, err
		}
	} else if err := consumeBatches(ctx, tx, accountID, -in.Amount, txnID); err != nil {
		return app.AdjustResult{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE point_account SET balance = ?, version = version + 1 WHERE id = ?`,
		after, accountID); err != nil {
		return app.AdjustResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return app.AdjustResult{}, err
	}
	return app.AdjustResult{AccountID: accountID, UserID: in.UserID, Balance: after, AuditID: auditID}, nil
}

// ListConfig 读取全部系统参数（UC-14：运营可调，改完立即生效，无需发版）。
func (s *Store) ListConfig(ctx context.Context) ([]app.ConfigItem, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT config_key, config_value, value_type, description FROM sys_config ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]app.ConfigItem, 0, 8)
	for rows.Next() {
		var it app.ConfigItem
		if err := rows.Scan(&it.Key, &it.Value, &it.ValueType, &it.Description); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// UpdateConfig 批量修改系统参数：键必须已存在，值按 value_type 与语义约束校验。
// 整批在一个事务里完成，任一项非法则全部回滚，避免"改一半"。
// 每个真正发生变化的参数都写一条 audit_log（BR-14：操作人/前后值/理由）。
// 返回更新后的全量参数，便于调用方直接刷新界面。
func (s *Store) UpdateConfig(ctx context.Context, operatorID int64, updates []app.ConfigUpdate) ([]app.ConfigItem, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("%w: 至少需要一项参数", app.ErrInvalidInput)
	}
	seen := make(map[string]bool, len(updates))
	normalized := make([]app.ConfigUpdate, 0, len(updates))
	for _, up := range updates {
		key := strings.TrimSpace(up.Key)
		value := strings.TrimSpace(up.Value)
		if key == "" || value == "" {
			return nil, fmt.Errorf("%w: key / value 不能为空", app.ErrInvalidInput)
		}
		if seen[key] {
			return nil, fmt.Errorf("%w: 参数键 %s 重复", app.ErrInvalidInput, key)
		}
		seen[key] = true
		normalized = append(normalized, app.ConfigUpdate{Key: key, Value: value})
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, up := range normalized {
		var configID int64
		var valueType, beforeValue string
		err := tx.QueryRowContext(ctx,
			`SELECT id, value_type, config_value FROM sys_config WHERE config_key = ? FOR UPDATE`, up.Key,
		).Scan(&configID, &valueType, &beforeValue)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: 未知参数键 %s", app.ErrInvalidInput, up.Key)
		}
		if err != nil {
			return nil, err
		}
		if err := app.ValidateConfigValue(up.Key, valueType, up.Value); err != nil {
			return nil, fmt.Errorf("%w: %s=%s", app.ErrInvalidInput, up.Key, up.Value)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE sys_config SET config_value = ? WHERE config_key = ?`, up.Value, up.Key); err != nil {
			return nil, err
		}
		// 值没变就不记审计，避免刷屏；变了才留痕。
		if beforeValue == up.Value {
			continue
		}
		if _, err := writeAudit(ctx, tx, auditEntry{
			operatorID:  operatorID,
			action:      "CONFIG_UPDATE",
			targetType:  "sys_config",
			targetID:    configID,
			targetKey:   up.Key,
			beforeValue: beforeValue,
			afterValue:  up.Value,
			reason:      "参数配置变更",
		}); err != nil {
			return nil, err
		}
	}
	if err := checkDistanceRange(ctx, tx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListConfig(ctx)
}

// checkDistanceRange 在事务内复核距离区间 min < max：
// 区间反了会让所有行程上报被拒（domain.PointsRule.CheckRule 会返回配置错误）。
func checkDistanceRange(ctx context.Context, tx *sql.Tx) error {
	read := func(key string) (float64, error) {
		var raw string
		err := tx.QueryRowContext(ctx,
			`SELECT config_value FROM sys_config WHERE config_key = ?`, key).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: 缺少参数 %s", app.ErrInvalidInput, key)
		}
		if err != nil {
			return 0, err
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %s 不是合法小数", app.ErrInvalidInput, key)
		}
		return v, nil
	}
	minKm, err := read("trip_min_distance_km")
	if err != nil {
		return err
	}
	maxKm, err := read("trip_max_distance_km")
	if err != nil {
		return err
	}
	if minKm >= maxKm {
		return fmt.Errorf("%w: trip_min_distance_km(%v) 必须小于 trip_max_distance_km(%v)",
			app.ErrInvalidInput, minKm, maxKm)
	}
	return nil
}

// UpdateTravelMode 更新出行方式的名称 / 排放因子 / 启用状态（UC-14）。
// 实际排放因子必须小于基准因子，否则该方式永远算不出碳减排量；变更写 audit_log（BR-14）。
func (s *Store) UpdateTravelMode(ctx context.Context, operatorID int64, code string, in app.TravelModeUpdate) (app.TravelMode, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return app.TravelMode{}, fmt.Errorf("%w: 出行方式代码不能为空", app.ErrInvalidInput)
	}
	if err := app.ValidateTravelModeUpdate(in); err != nil {
		return app.TravelMode{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.TravelMode{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var cur app.TravelMode
	var enabled int
	err = tx.QueryRowContext(ctx,
		`SELECT code, name, ef_base, ef_actual, enabled FROM travel_mode WHERE code = ? FOR UPDATE`, code,
	).Scan(&cur.Code, &cur.Name, &cur.EFBase, &cur.EFActual, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return app.TravelMode{}, app.ErrNotFound
	}
	if err != nil {
		return app.TravelMode{}, err
	}
	cur.Enabled = enabled != 0
	before := describeTravelMode(cur)

	if in.Name != nil {
		cur.Name = strings.TrimSpace(*in.Name)
	}
	if in.EFBase != nil {
		cur.EFBase = *in.EFBase
	}
	if in.EFActual != nil {
		cur.EFActual = *in.EFActual
	}
	if in.Enabled != nil {
		cur.Enabled = *in.Enabled
	}
	if cur.EFActual >= cur.EFBase {
		return app.TravelMode{}, fmt.Errorf("%w: efActual(%v) 必须小于 efBase(%v)",
			app.ErrInvalidInput, cur.EFActual, cur.EFBase)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE travel_mode SET name = ?, ef_base = ?, ef_actual = ?, enabled = ?
		 WHERE code = ?`, cur.Name, cur.EFBase, cur.EFActual, cur.Enabled, cur.Code); err != nil {
		return app.TravelMode{}, err
	}
	if after := describeTravelMode(cur); after != before {
		if _, err := writeAudit(ctx, tx, auditEntry{
			operatorID:  operatorID,
			action:      "TRAVEL_MODE_UPDATE",
			targetType:  "travel_mode",
			targetKey:   cur.Code,
			beforeValue: before,
			afterValue:  after,
			reason:      "出行方式参数变更",
		}); err != nil {
			return app.TravelMode{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return app.TravelMode{}, err
	}
	return cur, nil
}

// describeTravelMode 把出行方式的可审计字段拼成一行，供 audit_log 的前后值使用。
func describeTravelMode(m app.TravelMode) string {
	return fmt.Sprintf("name=%s;efBase=%s;efActual=%s;enabled=%t",
		m.Name,
		strconv.FormatFloat(m.EFBase, 'f', -1, 64),
		strconv.FormatFloat(m.EFActual, 'f', -1, 64),
		m.Enabled)
}

// getProductAny 读取任意状态的商品（含 SKU 列表），供后台使用。
func (s *Store) getProductAny(ctx context.Context, id int64) (app.Product, error) {
	var p app.Product
	err := s.db.QueryRowContext(ctx,
		`SELECT id, merchant_id, name, description, category, cover_url, status, limit_per_user
		 FROM product WHERE id = ?`, id,
	).Scan(&p.ID, &p.MerchantID, &p.Name, &p.Description, &p.Category, &p.ImageURL, &p.Status, &p.LimitPerUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Product{}, app.ErrNotFound
		}
		return app.Product{}, err
	}
	p.Active = p.Status == "ONLINE"

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, sku_name, points_price, stock, sold_count
		 FROM product_sku WHERE product_id = ? ORDER BY id`, id)
	if err != nil {
		return app.Product{}, err
	}
	defer rows.Close()
	first := true
	for rows.Next() {
		var sku app.SKU
		if err := rows.Scan(&sku.ID, &sku.Name, &sku.PointsPrice, &sku.Stock, &sku.SoldCount); err != nil {
			return app.Product{}, err
		}
		if first || sku.PointsPrice < p.PointsPrice {
			p.PointsPrice = sku.PointsPrice
			first = false
		}
		p.Stock += sku.Stock
		p.SKUs = append(p.SKUs, sku)
	}
	return p, rows.Err()
}

func validProductStatus(s string) bool {
	switch s {
	case "DRAFT", "ONLINE", "OFFLINE":
		return true
	default:
		return false
	}
}
