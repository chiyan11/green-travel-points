import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { getSummary } from '@/services/points'
import { listOrders } from '@/services/orders'
import { getProduct } from '@/services/products'
import { okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'

/**
 * 枚举兜底：后端加了前端还不认识的状态时，必须「原样展示 + 稳定降级」，
 * 不能靠断言把新值说成某个已知状态，也不能把商品默认成可兑换。
 */
describe('枚举兜底', () => {
  it('未知订单状态落到 UNKNOWN 并保留原始值', async () => {
    server.use(
      http.get('*/api/v1/orders', () =>
        okJson(
          pageOf([
            {
              id: 1,
              userId: 'u_1005',
              orderNo: 'RO1',
              productId: 1001,
              skuId: 1,
              productName: '环保帆布袋',
              skuName: '标准款',
              pointsSpent: 100,
              quantity: 1,
              receiver: '本人',
              status: 'REFUNDING',
              redeemCode: 'ABC123',
              createdAt: '2026-11-24T10:00:00Z',
              updatedAt: '2026-11-24T10:00:00Z',
            },
          ]),
        ),
      ),
    )

    const { items } = await listOrders()
    const first = items[0]!
    expect(first.status).toBe('UNKNOWN')
    expect(first.statusText).toBe('REFUNDING')
  })

  it('缺失订单状态不再冒充 PENDING', async () => {
    server.use(
      http.get('*/api/v1/orders', () =>
        okJson(
          pageOf([
            {
              id: 2,
              userId: 'u_1005',
              orderNo: 'RO2',
              productId: 1001,
              skuId: 1,
              productName: '环保帆布袋',
              skuName: '标准款',
              pointsSpent: 100,
              quantity: 1,
              receiver: '本人',
              redeemCode: 'ABC123',
              createdAt: '2026-11-24T10:00:00Z',
              updatedAt: '2026-11-24T10:00:00Z',
            },
          ]),
        ),
      ),
    )

    const { items } = await listOrders()
    const first = items[0]!
    expect(first.status).toBe('UNKNOWN')
    expect(first.statusText).toBe('')
  })

  it('未知商品状态按已下架处理，不默认成可兑换', async () => {
    server.use(
      http.get('*/api/v1/products/1', () =>
        okJson({
          id: 1,
          name: '环保帆布袋',
          description: '',
          category: '文创',
          imageUrl: '',
          pointsPrice: 100,
          stock: 10,
          active: true,
          limitPerUser: 2,
          status: 'COMING_SOON',
        }),
      ),
    )

    const product = await getProduct('1')
    expect(product.status).toBe('OFFLINE')
  })

  it('dailyCap 为 0 表示不限上限，不被 || 兜成 500', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () =>
        okJson({
          balance: 150,
          frozen: 0,
          available: 150,
          totalCarbonSavedGrams: 15100,
          earnedThisMonth: 150,
          earnedToday: 150,
          dailyCap: 0,
        }),
      ),
    )

    const summary = await getSummary()
    expect(summary.dailyCap).toBe(0)
  })

  it('未知流水类型保留原始串，不被硬塞成 audit_log', async () => {
    server.use(
      http.get('*/api/v1/points/transactions', () =>
        okJson(
          pageOf([
            {
              id: 3,
              type: 'BONUS',
              amount: 20,
              balanceAfter: 170,
              reason: '',
              refType: 'campaign',
              refId: 7,
              createdAt: '2026-11-24T10:00:00Z',
            },
          ]),
        ),
      ),
    )

    const { listTxns } = await import('@/services/points')
    const { items } = await listTxns()
    const txn = items[0]!
    expect(txn.type).toBe('UNKNOWN')
    expect(txn.typeText).toBe('BONUS')
    expect(txn.bizRefType).toBe('UNKNOWN')
    expect(txn.bizRefTypeText).toBe('campaign')
  })
})
