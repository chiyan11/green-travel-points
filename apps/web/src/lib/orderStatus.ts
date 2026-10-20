import type { OrderStatus, PointTxnType } from '@/types/domain'

/**
 * 订单状态机元数据。口径依据 docs/项目分工与迭代计划.md §2.3：
 * PENDING → PAID → SHIPPED → COMPLETED，任意非终态可 CANCELLED（原型先仅开放 PENDING 取消）。
 */
export const ORDER_STATUS_META: Record<
  OrderStatus,
  { label: string; badge: string; cancellable: boolean }
> = {
  PENDING: { label: '待支付', badge: 'badge--muted', cancellable: true },
  PAID: { label: '已支付 · 待发货', badge: 'badge--info', cancellable: false },
  SHIPPED: { label: '已发货', badge: 'badge--warn', cancellable: false },
  COMPLETED: { label: '已完成', badge: 'badge--ok', cancellable: false },
  CANCELLED: { label: '已取消', badge: 'badge--danger', cancellable: false },
}

/** 合法跃迁边表；非法跃迁必须被后端拒绝，前端仅做按钮禁用提示。 */
const TRANSITIONS: Record<OrderStatus, readonly OrderStatus[]> = {
  PENDING: ['PAID', 'CANCELLED'],
  PAID: ['SHIPPED', 'CANCELLED'],
  SHIPPED: ['COMPLETED'],
  COMPLETED: [],
  CANCELLED: [],
}

export function canTransition(from: OrderStatus, to: OrderStatus): boolean {
  return TRANSITIONS[from].includes(to)
}

export const POINT_TXN_TYPE_META: Record<PointTxnType, { label: string; sign: 1 | -1 }> = {
  EARN: { label: '出行获得', sign: 1 },
  SPEND: { label: '商城消耗', sign: -1 },
  EXPIRE: { label: '过期扣减', sign: -1 },
  ADJUST: { label: '人工调整', sign: 1 },
  REVERSE: { label: '冲正返还', sign: 1 },
}
