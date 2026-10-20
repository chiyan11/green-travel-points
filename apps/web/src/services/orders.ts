/**
 * 兑换订单服务（MOCK）。
 * TODO 联调替换：GET /api/v1/orders、POST /api/v1/orders、POST /api/v1/orders/{id}/cancel。
 * 防超卖、扣减与冻结以后端事务为准，前端仅做预校验与提示。
 */
import { mockDb, nextId } from '@/mock/data'
import type { RedeemOrder } from '@/types/domain'

export function listOrders(): RedeemOrder[] {
  return mockDb.getOrders()
}

export function createOrder(productId: string, receiver: string): RedeemOrder {
  const product = mockDb.getProduct(productId)
  if (!product) throw new Error('商品不存在或已下架')
  if (product.stock <= 0) throw new Error('库存不足，无法兑换')
  if (mockDb.getBalance() < product.pointsPrice) throw new Error('积分余额不足')

  const now = new Date().toISOString()
  const order: RedeemOrder = {
    id: nextId('order'),
    productId,
    productName: product.name,
    pointsSpent: product.pointsPrice,
    status: 'PENDING',
    receiver: receiver.trim() || '本人',
    createdAt: now,
    updatedAt: now,
  }
  mockDb.addOrder(order)
  mockDb.addPoints(-product.pointsPrice)
  mockDb.decrementStock(productId)
  mockDb.addTxn({
    id: nextId('txn'),
    type: 'SPEND',
    amount: -product.pointsPrice,
    balanceAfter: mockDb.getBalance(),
    reason: `兑换${product.name}`,
    refType: 'ORDER',
    refId: order.id,
    createdAt: now,
  })
  return order
}

export function cancelOrder(id: string): void {
  const order = mockDb.getOrders().find((o) => o.id === id)
  if (!order) throw new Error('订单不存在')
  if (order.status !== 'PENDING') throw new Error('仅待支付订单可以取消')

  const product = mockDb.getProduct(order.productId)
  mockDb.updateOrder(id, { status: 'CANCELLED' })
  mockDb.addPoints(order.pointsSpent)
  if (product) mockDb.restoreStock(product.id)
  mockDb.addTxn({
    id: nextId('txn'),
    type: 'REVERSE',
    amount: order.pointsSpent,
    balanceAfter: mockDb.getBalance(),
    reason: `取消订单 ${order.id}，积分返还`,
    refType: 'ORDER',
    refId: id,
    createdAt: new Date().toISOString(),
  })
}
