/**
 * 兑换订单服务。
 * GET /api/v1/orders、POST /api/v1/orders、POST /api/v1/orders/{id}/cancel（需登录）。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成）。
 * 扣减、冻结、库存一致性以后端事务为准，前端不再本地改余额/库存。
 */
import { api } from '@/lib/http'
import type { Order as WireOrder, OrderPage } from '@/types/api'
import type { OrderStatus, RedeemOrder } from '@/types/domain'

function toOrder(r: WireOrder): RedeemOrder {
  return {
    id: String(r.id),
    orderNo: r.orderNo,
    productId: String(r.productId),
    skuId: r.skuId === 0 ? '' : String(r.skuId),
    skuName: r.skuName,
    productName: r.productName,
    pointsSpent: r.pointsSpent,
    quantity: r.quantity,
    receiver: r.receiver,
    status: (r.status || 'PENDING') as OrderStatus,
    redeemCode: r.redeemCode,
    createdAt: r.createdAt,
    updatedAt: r.updatedAt,
  }
}

export async function listOrders(): Promise<RedeemOrder[]> {
  const page = await api.get<OrderPage>('/api/v1/orders?page=1&pageSize=50')
  return (page.items ?? []).map(toOrder)
}

/**
 * 兑换下单。idempotencyKey 由调用方为「一次兑换意图」生成并复用：
 * 网络失败后重试必须带同一个 key，否则会重复扣分；兑换成功或用户放弃后应丢弃。
 */
export async function createOrder(
  productId: string,
  receiver: string,
  idempotencyKey: string,
): Promise<RedeemOrder> {
  const raw = await api.post<WireOrder>(
    '/api/v1/orders',
    { productId: Number(productId), receiver: receiver.trim() || '本人' },
    { 'Idempotency-Key': idempotencyKey },
  )
  return toOrder(raw)
}

export async function cancelOrder(id: string): Promise<void> {
  await api.post(`/api/v1/orders/${encodeURIComponent(id)}/cancel`, { reason: '用户在前端取消' })
}
