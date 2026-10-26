package repo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"greentravelpoints/internal/app"
)

// orderColumns 是订单查询的固定列顺序，scanOrder 依赖该顺序。
const orderColumns = `id, order_no, product_id, sku_id, product_name, sku_name,
	points_spent, quantity, receiver, status, redeem_code, idempotency_key,
	created_at, updated_at, shipped_at, completed_at, cancelled_at, cancel_reason`

// CreateOrder 兑换下单（M7）。
//
// 事务步骤（对应 docs/uml/09-序列图-兑换下单.puml）：
//  1. 幂等键命中 → 直接返回原订单（BR-15）；
//  2. 账户行 FOR UPDATE → SKU 行 FOR UPDATE（固定顺序，避免死锁）；
//  3. 锁后复核：商品在线、库存 > 0、balance - frozen ≥ 价格、未超限兑（BR-09）；
//  4. 写订单 + REDEEM 流水 + 扣余额 + 扣库存 + 消耗积分批次；
//  5. 写 event_outbox(OrderPaid)，提交。
func (s *Store) CreateOrder(ctx context.Context, in app.CreateOrderInput) (app.Order, bool, error) {
	if in.IdempotencyKey != "" {
		o, err := s.findOrderByIdem(ctx, in.UserID, in.IdempotencyKey)
		if err == nil {
			return o, true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, false, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.Order{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	// 账户不存在时先建空账户，保证后续可以行锁。
	if _, err := tx.ExecContext(ctx,
		`INSERT IGNORE INTO point_account (user_id) VALUES (?)`, in.UserID); err != nil {
		return app.Order{}, false, err
	}
	var accountID, balance, frozen int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id, balance, frozen FROM point_account WHERE user_id = ? FOR UPDATE`, in.UserID,
	).Scan(&accountID, &balance, &frozen); err != nil {
		return app.Order{}, false, err
	}

	// SKU 行锁：指定 skuId 就锁它，否则挑第一个有库存的 SKU。
	var skuID, productID, price int64
	var skuName string
	var stock int
	if in.SKUId > 0 {
		err = tx.QueryRowContext(ctx,
			`SELECT id, product_id, sku_name, points_price, stock
			 FROM product_sku WHERE id = ? FOR UPDATE`, in.SKUId,
		).Scan(&skuID, &productID, &skuName, &price, &stock)
	} else {
		err = tx.QueryRowContext(ctx,
			`SELECT id, product_id, sku_name, points_price, stock
			 FROM product_sku WHERE product_id = ? AND stock > 0
			 ORDER BY id ASC LIMIT 1 FOR UPDATE`, in.ProductID,
		).Scan(&skuID, &productID, &skuName, &price, &stock)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, false, app.ErrOutOfStock
		}
		return app.Order{}, false, err
	}
	if in.ProductID > 0 && productID != in.ProductID {
		return app.Order{}, false, app.ErrNotFound
	}

	// 锁后复核：商品必须在线。
	var productName, productStatus string
	var limitPerUser int
	if err := tx.QueryRowContext(ctx,
		`SELECT name, status, limit_per_user FROM product WHERE id = ?`, productID,
	).Scan(&productName, &productStatus, &limitPerUser); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, false, app.ErrProductOffline
		}
		return app.Order{}, false, err
	}
	if productStatus != "ONLINE" {
		return app.Order{}, false, app.ErrProductOffline
	}
	if stock <= 0 {
		return app.Order{}, false, app.ErrOutOfStock
	}
	if balance-frozen < price {
		return app.Order{}, false, app.ErrInsufficientPoints
	}
	if limitPerUser > 0 {
		var used int64
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM redeem_order
			 WHERE user_id = ? AND product_id = ? AND status <> 'CANCELLED'`,
			in.UserID, productID).Scan(&used); err != nil {
			return app.Order{}, false, err
		}
		if used >= int64(limitPerUser) {
			return app.Order{}, false, app.ErrLimitExceeded
		}
	}

	newBalance := balance - price
	now := time.Now()
	orderNo := newOrderNo(now)
	redeemCode := newRedeemCode()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO redeem_order
		   (order_no, user_id, product_id, sku_id, product_name, sku_name,
		    points_spent, receiver, status, redeem_code, idempotency_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)`,
		orderNo, in.UserID, productID, skuID, productName, skuName,
		price, in.Receiver, redeemCode, nullableString(in.IdempotencyKey))
	if err != nil {
		// 并发下同一幂等键只会有一个成功，另一个回读原订单（BR-15）。
		if isDuplicate(err) && in.IdempotencyKey != "" {
			_ = tx.Rollback()
			if o, ferr := s.findOrderByIdem(ctx, in.UserID, in.IdempotencyKey); ferr == nil {
				return o, true, nil
			}
		}
		return app.Order{}, false, err
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return app.Order{}, false, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO point_txn
		   (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id, occurred_at)
		 VALUES (?, 'REDEEM', ?, ?, 'redeem_order', ?, ?)`,
		accountID, -price, newBalance, orderID, now); err != nil {
		return app.Order{}, false, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE point_account SET balance = ?, version = version + 1 WHERE id = ?`,
		newBalance, accountID); err != nil {
		return app.Order{}, false, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE product_sku SET stock = stock - 1, sold_count = sold_count + 1 WHERE id = ?`,
		skuID); err != nil {
		return app.Order{}, false, err
	}
	if err := consumeBatches(ctx, tx, accountID, price, now); err != nil {
		return app.Order{}, false, err
	}
	if err := insertOutbox(ctx, tx, "OrderPaid", "order", orderNo, map[string]any{
		"orderNo":   orderNo,
		"userId":    in.UserID,
		"skuId":     skuID,
		"points":    price,
		"productId": productID,
	}); err != nil {
		return app.Order{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return app.Order{}, false, err
	}
	o, err := s.getOrder(ctx, orderID)
	return o, false, err
}

// ListOrders 分页返回当前用户的兑换订单（M7）。
func (s *Store) ListOrders(ctx context.Context, q app.OrderQuery) (app.Page[app.Order], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)

	where := []string{"user_id = ?"}
	args := []any{q.UserID}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, strings.ToUpper(q.Status))
	}
	whereSQL := joinAnd(where)

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

// CancelOrder 取消订单（M7 / BR-12）：补 REFUND 反向流水、退余额、回补库存、
// 生成退款积分批次，并把订单置为 CANCELLED。仅 PENDING 可取消。
func (s *Store) CancelOrder(ctx context.Context, userID, orderID int64, reason string) (app.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.Order{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var status, orderNo string
	var points, skuID int64
	if err := tx.QueryRowContext(ctx,
		`SELECT status, order_no, points_spent, sku_id FROM redeem_order
		 WHERE id = ? AND user_id = ? FOR UPDATE`, orderID, userID,
	).Scan(&status, &orderNo, &points, &skuID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, app.ErrNotFound
		}
		return app.Order{}, err
	}
	if status != "PENDING" {
		return app.Order{}, app.ErrIllegalTransition
	}

	var accountID, balance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id, balance FROM point_account WHERE user_id = ? FOR UPDATE`, userID,
	).Scan(&accountID, &balance); err != nil {
		return app.Order{}, err
	}

	now := time.Now()
	newBalance := balance + points
	res, err := tx.ExecContext(ctx,
		`INSERT INTO point_txn
		   (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id, occurred_at)
		 VALUES (?, 'REFUND', ?, ?, 'redeem_order', ?, ?)`,
		accountID, points, newBalance, orderID, now)
	if err != nil {
		return app.Order{}, err
	}
	refundTxnID, err := res.LastInsertId()
	if err != nil {
		return app.Order{}, err
	}

	// 退款积分作为新批次，按 points_expire_days 重新计有效期（BR-11）。
	days := s.pointsExpireDays(ctx)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO point_batch
		   (account_id, grant_txn_id, granted_amount, remaining, expires_at, status)
		 VALUES (?, ?, ?, ?, DATE_ADD(NOW(3), INTERVAL ? DAY), 'ACTIVE')`,
		accountID, refundTxnID, points, points, days); err != nil {
		return app.Order{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE point_account SET balance = ?, version = version + 1 WHERE id = ?`,
		newBalance, accountID); err != nil {
		return app.Order{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE product_sku SET stock = stock + 1, sold_count = GREATEST(sold_count - 1, 0) WHERE id = ?`,
		skuID); err != nil {
		return app.Order{}, err
	}
	cancelReason := strings.TrimSpace(reason)
	if cancelReason == "" {
		cancelReason = "用户取消"
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE redeem_order SET status = 'CANCELLED', cancelled_at = NOW(3), cancel_reason = ?
		 WHERE id = ?`, cancelReason, orderID); err != nil {
		return app.Order{}, err
	}
	if err := insertOutbox(ctx, tx, "OrderCancelled", "order", orderNo, map[string]any{
		"orderNo": orderNo,
		"userId":  userID,
		"points":  points,
		"reason":  cancelReason,
	}); err != nil {
		return app.Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return app.Order{}, err
	}
	return s.getOrder(ctx, orderID)
}

func (s *Store) getOrder(ctx context.Context, id int64) (app.Order, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM redeem_order WHERE id = ?`, id)
	o, err := scanOrder(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Order{}, app.ErrNotFound
		}
		return app.Order{}, err
	}
	return o, nil
}

func (s *Store) findOrderByIdem(ctx context.Context, userID int64, key string) (app.Order, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+orderColumns+` FROM redeem_order WHERE user_id = ? AND idempotency_key = ?`,
		userID, key)
	return scanOrder(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row rowScanner) (app.Order, error) {
	var (
		o           app.Order
		idem        sql.NullString
		shippedAt   sql.NullTime
		completedAt sql.NullTime
		cancelledAt sql.NullTime
	)
	if err := row.Scan(&o.ID, &o.OrderNo, &o.ProductID, &o.SKUId, &o.ProductName, &o.SKUName,
		&o.PointsSpent, &o.Quantity, &o.Receiver, &o.Status, &o.RedeemCode, &idem,
		&o.CreatedAt, &o.UpdatedAt, &shippedAt, &completedAt, &cancelledAt, &o.CancelReason); err != nil {
		return app.Order{}, err
	}
	if shippedAt.Valid {
		o.ShippedAt = &shippedAt.Time
	}
	if completedAt.Valid {
		o.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		o.CancelledAt = &cancelledAt.Time
	}
	return o, nil
}

// consumeBatches 按先进先出扣减积分批次（BR-11）。
// 批次不足（历史数据未补批次）时不报错，以账户余额为准。
func consumeBatches(ctx context.Context, tx *sql.Tx, accountID, need int64, now time.Time) error {
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
	}
	return nil
}

func insertOutbox(ctx context.Context, tx *sql.Tx, topic, aggregateType, aggregateID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		 VALUES (?, ?, ?, ?)`, topic, aggregateType, aggregateID, raw)
	return err
}

func (s *Store) pointsExpireDays(ctx context.Context) int64 {
	var raw string
	if err := s.db.QueryRowContext(ctx,
		`SELECT config_value FROM sys_config WHERE config_key = 'points_expire_days'`,
	).Scan(&raw); err != nil {
		return 365
	}
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n <= 0 {
		return 365
	}
	return n
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// newOrderNo 生成订单号：RO + yyyyMMddHHmmss + 4 位随机数（20 位，< varchar(32)）。
func newOrderNo(now time.Time) string {
	var buf [2]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("RO%s%04d", now.Format("20060102150405"), binary.BigEndian.Uint16(buf[:])%10000)
}

// newRedeemCode 生成 10 位核销码（去掉易混淆的 0/O/1/I）。
func newRedeemCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var buf [10]byte
	_, _ = rand.Read(buf[:])
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out)
}
