/**
 * 兑换订单服务（MOCK）。
 * TODO 联调替换：GET /api/v1/orders、POST /api/v1/orders、POST /api/v1/orders/{id}/cancel。
 * 防超卖、扣减与冻结以后端事务为准，前端仅做预校验与提示。
 * 字段口径对齐 redeem_order（000005）：orderNo/redeemCode/quantity 等。
 */
import { mockDb, nextId } from '@/mock/data'
import type { RedeemOrder } from '@/types/domain'

export function listOrders(): RedeemOrder[] {
  return mockDb.getOrders()
}

function nextOrderNo(): string {
  const d = new Date()
  const ymd = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, '0')}${String(d.getDate()).padStart(2, '0')}`
  return `SO${ymd}${String(Math.floor(Math.random() * 9000) + 1000)}`
}

export function createOrder(productId: string, receiver: string): RedeemOrder {
  const product = mockDb.getProduct(productId)
  if (!product) throw new Error('商品不存在或已下架')
  if (product.status !== 'ONLINE') throw new Error('商品已下架，无法兑换')
  if (product.stock <= 0) throw new Error('库存不足，无法兑换')
  if (mockDb.getBalance() < product.pointsPrice) throw new Error('积分余额不足')

  const now = new Date().toISOString()
  const order: RedeemOrder = {
    id: nextId('order'),
    orderNo: nextOrderNo(),
    productId,
    skuId: `${productId}_sku`,
    skuName: '默认规格',
    productName: product.name,
    pointsSpent: product.pointsPrice,
    quantity: 1,
    receiver: receiver.trim() || '本人',
    status: 'PENDING',
    redeemCode: '',
    createdAt: now,
    updatedAt: now,
  }
  mockDb.addOrder(order)
  mockDb.addPoints(-product.pointsPrice)
  mockDb.decrementStock(productId)
  mockDb.addTxn({
    id: nextId('txn'),
    type: 'REDEEM',
    amount: -product.pointsPrice,
    balanceAfter: mockDb.getBalance(),
    bizRefType: 'redeem_order',
    bizRefId: order.id,
    occurredAt: now,
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
    type: 'REFUND',
    amount: order.pointsSpent,
    balanceAfter: mockDb.getBalance(),
    bizRefType: 'redeem_order',
    bizRefId: id,
    occurredAt: new Date().toISOString(),
  })
}
