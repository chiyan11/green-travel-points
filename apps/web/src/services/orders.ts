/**
 * 兑换订单服务（联调版）。
 * GET /api/v1/orders、POST /api/v1/orders、POST /api/v1/orders/{id}/cancel（需 X-User-Id）。
 * 扣减、冻结、库存一致性以后端事务为准，前端不再本地改余额/库存。
 */
import { api } from '@/lib/http'
import type { OrderStatus, Page, RedeemOrder } from '@/types/domain'

interface RawOrder {
  id: number
  orderNo: string
  productId: number
  skuId?: number
  productName: string
  skuName?: string
  pointsSpent: number
  quantity?: number
  receiver?: string
  status: string
  redeemCode?: string
  createdAt: string
  updatedAt: string
}

function toOrder(r: RawOrder): RedeemOrder {
  return {
    id: String(r.id),
    orderNo: r.orderNo,
    productId: String(r.productId),
    skuId: String(r.skuId ?? ''),
    skuName: r.skuName ?? '',
    productName: r.productName,
    pointsSpent: r.pointsSpent,
    quantity: r.quantity ?? 1,
    receiver: r.receiver ?? '本人',
    status: (r.status || 'PENDING') as OrderStatus,
    redeemCode: r.redeemCode ?? '',
    createdAt: r.createdAt,
    updatedAt: r.updatedAt,
  }
}

export async function listOrders(): Promise<RedeemOrder[]> {
  const page = await api.get<Page<RawOrder>>('/api/v1/orders?page=1&pageSize=50')
  return (page.items ?? []).map(toOrder)
}

export async function createOrder(productId: string, receiver: string): Promise<RedeemOrder> {
  const raw = await api.post<RawOrder>(
    '/api/v1/orders',
    { productId: Number(productId), receiver: receiver.trim() || '本人' },
    { 'Idempotency-Key': crypto.randomUUID() },
  )
  return toOrder(raw)
}

export async function cancelOrder(id: string): Promise<void> {
  await api.post(`/api/v1/orders/${encodeURIComponent(id)}/cancel`, { reason: '用户在前端取消' })
}
