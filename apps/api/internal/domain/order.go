package domain

import (
	"errors"
	"fmt"
)

// OrderStatus 是兑换订单状态，取值必须与 docs/uml/10-状态图-订单.puml、
// db/migrations/000005_mall_orders.up.sql 的 redeem_order.status 枚举一一对应。
type OrderStatus string

const (
	OrderPending   OrderStatus = "PENDING"   // 待商户确认（已扣分、已占库存）
	OrderPaid      OrderStatus = "PAID"      // 商户已确认，待发货
	OrderShipped   OrderStatus = "SHIPPED"   // 已发货，核销码生效
	OrderCompleted OrderStatus = "COMPLETED" // 已核销 / 已确认收货
	OrderCancelled OrderStatus = "CANCELLED" // 已取消，积分原路退回
)

// ErrIllegalOrderTransition 表示试图执行状态机不允许的跃迁。
var ErrIllegalOrderTransition = errors.New("domain: 非法的订单状态跃迁")

// orderTransitions 是状态机允许的边。终态（COMPLETED / CANCELLED）不可再变。
var orderTransitions = map[OrderStatus]map[OrderStatus]bool{
	OrderPending: {OrderPaid: true, OrderCancelled: true},
	OrderPaid:    {OrderShipped: true, OrderCancelled: true},
	OrderShipped: {OrderCompleted: true, OrderCancelled: true},
}

// CanTransition 判断 from → to 是否为合法跃迁。
func CanTransition(from, to OrderStatus) bool {
	nexts, ok := orderTransitions[from]
	if !ok {
		return false
	}
	return nexts[to]
}

// CheckOrderTransition 在跃迁非法时返回 ErrIllegalOrderTransition。
func CheckOrderTransition(from, to OrderStatus) error {
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s → %s", ErrIllegalOrderTransition, from, to)
}
