import type { OrderStatus, PointTxnType } from '@/types/domain'

/**
 * 订单状态机元数据。口径见 docs/api/README.md §3.1：
 * PENDING → PAID → SHIPPED → COMPLETED，任意非终态可 CANCELLED（用户端仅 PENDING 可取消）。
 *
 * 注意：积分兑换**没有支付环节**——下单事务里已经扣分并占住库存，
 * 所以 PENDING 的文案是「待商户确认」，不要写「待支付」。
 */
export const ORDER_STATUS_META: Record<
  OrderStatus,
  { label: string; badgeVariant: 'neutral' | 'info' | 'success' | 'warning' | 'error'; cancellable: boolean }
> = {
  PENDING: { label: '待商户确认', badgeVariant: 'warning', cancellable: true },
  PAID: { label: '已确认 · 待发货', badgeVariant: 'info', cancellable: false },
  SHIPPED: { label: '已发货', badgeVariant: 'success', cancellable: false },
  COMPLETED: { label: '已完成', badgeVariant: 'neutral', cancellable: false },
  CANCELLED: { label: '已取消', badgeVariant: 'error', cancellable: false },
  UNKNOWN: { label: '未知状态', badgeVariant: 'neutral', cancellable: false },
}

/** 旧式 `.badge--*` 类名（运营后台仍在用 span.badge，不依赖 Astryx Badge 组件）。 */
const BADGE_CLASS: Record<(typeof ORDER_STATUS_META)[OrderStatus]['badgeVariant'], string> = {
  neutral: 'badge--muted',
  info: 'badge--info',
  warning: 'badge--warn',
  success: 'badge--ok',
  error: 'badge--danger',
}

export function orderStatusBadgeClass(status: OrderStatus): string {
  return BADGE_CLASS[ORDER_STATUS_META[status].badgeVariant]
}

/**
 * 展示用文案：后端出现了前端还不认识的枚举时，直接显示原始值，
 * 而不是断言成某个已知状态（那会把「新状态」说成「待商户确认」）。
 */
export function orderStatusLabel(status: OrderStatus, raw?: string): string {
  if (status !== 'UNKNOWN') return ORDER_STATUS_META[status].label
  const text = (raw ?? '').trim()
  return text !== '' ? text : ORDER_STATUS_META.UNKNOWN.label
}

/** 合法跃迁边表；非法跃迁必须被后端拒绝，前端仅做按钮禁用提示。 */
const TRANSITIONS: Record<OrderStatus, readonly OrderStatus[]> = {
  PENDING: ['PAID', 'CANCELLED'],
  PAID: ['SHIPPED', 'CANCELLED'],
  SHIPPED: ['COMPLETED'],
  COMPLETED: [],
  CANCELLED: [],
  UNKNOWN: [],
}

export function canTransition(from: OrderStatus, to: OrderStatus): boolean {
  return TRANSITIONS[from].includes(to)
}

/** 键名与 point_txn.txn_type（000003）一致：TRAVEL/REDEEM/REFUND/ADJUST/EXPIRE。 */
export const POINT_TXN_TYPE_META: Record<PointTxnType, { label: string; sign: 1 | -1 }> = {
  TRAVEL: { label: '出行获得', sign: 1 },
  REDEEM: { label: '商城消耗', sign: -1 },
  REFUND: { label: '取消返还', sign: 1 },
  EXPIRE: { label: '过期扣减', sign: -1 },
  ADJUST: { label: '人工调整', sign: 1 },
  UNKNOWN: { label: '其他变动', sign: 1 },
}

/** 流水类型文案同样保留原始值兜底。 */
export function pointTxnTypeLabel(type: PointTxnType, raw?: string): string {
  if (type !== 'UNKNOWN') return POINT_TXN_TYPE_META[type].label
  const text = (raw ?? '').trim()
  return text !== '' ? text : POINT_TXN_TYPE_META.UNKNOWN.label
}
