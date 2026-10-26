package domain

import "testing"

func TestCanTransition_LegalEdges(t *testing.T) {
	cases := [][2]OrderStatus{
		{OrderPending, OrderPaid},
		{OrderPending, OrderCancelled},
		{OrderPaid, OrderShipped},
		{OrderPaid, OrderCancelled},
		{OrderShipped, OrderCompleted},
		{OrderShipped, OrderCancelled},
	}
	for _, c := range cases {
		if !CanTransition(c[0], c[1]) {
			t.Errorf("期望 %s → %s 合法，实际被拒绝", c[0], c[1])
		}
	}
}

func TestCanTransition_IllegalEdges(t *testing.T) {
	cases := [][2]OrderStatus{
		{OrderPending, OrderShipped},   // 跳过商户确认
		{OrderPending, OrderCompleted}, // 跳两跳
		{OrderPaid, OrderPending},
		{OrderShipped, OrderPaid}, // 不可回退
		{OrderCompleted, OrderCancelled},
		{OrderCancelled, OrderPending},
		{OrderCompleted, OrderCompleted},
	}
	for _, c := range cases {
		if CanTransition(c[0], c[1]) {
			t.Errorf("期望 %s → %s 非法，实际被允许", c[0], c[1])
		}
		if err := CheckOrderTransition(c[0], c[1]); err == nil {
			t.Errorf("期望 %s → %s 返回错误", c[0], c[1])
		}
	}
}

func TestCheckOrderTransition_LegalReturnsNil(t *testing.T) {
	if err := CheckOrderTransition(OrderPending, OrderPaid); err != nil {
		t.Fatalf("合法跃迁不应报错: %v", err)
	}
}
