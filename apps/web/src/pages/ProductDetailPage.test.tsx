import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ProductDetailPage } from '@/pages/ProductDetailPage'
import { renderPage } from '@/test/render'
import { failJson, okJson } from '@/test/http'
import { server } from '@/test/server'
import type { PointsSummary, Product } from '@/types/api'

function makeProduct(over: Partial<Product> = {}): Product {
  return {
    id: 1,
    name: '环保帆布袋',
    description: '再生棉麻材质',
    category: '生活',
    imageUrl: '',
    pointsPrice: 100,
    stock: 49,
    active: true,
    limitPerUser: 0,
    ...over,
  }
}

function makeSummary(over: Partial<PointsSummary> = {}): PointsSummary {
  return {
    balance: 1000,
    frozen: 0,
    available: 1000,
    dailyCap: 500,
    earnedThisMonth: 120,
    earnedToday: 40,
    totalCarbonSavedGrams: 25600,
    ...over,
  }
}

function makeOrder(over: Record<string, unknown> = {}) {
  return {
    id: 7,
    userId: 1005,
    orderNo: 'GT202611240001',
    productId: 1,
    skuId: 1,
    productName: '环保帆布袋',
    skuName: '默认',
    pointsSpent: 100,
    quantity: 1,
    receiver: '本人',
    status: 'PAID',
    redeemCode: 'RC-0007',
    createdAt: '2026-11-24T02:00:00Z',
    updatedAt: '2026-11-24T02:00:00Z',
    ...over,
  }
}

function renderDetail() {
  return renderPage(<ProductDetailPage />, { route: '/mall/1', path: '/mall/:id' })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ProductDetailPage', () => {
  it('库存为 0 时兑换按钮禁用并显示已兑完', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct({ stock: 0 }))),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
    )
    renderDetail()

    const button = await screen.findByRole('button', { name: '已兑完' })
    expect(button).toBeDisabled()
  })

  it('余额不足时禁用兑换并提示差额', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct({ pointsPrice: 500 }))),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary({ balance: 120 }))),
    )
    renderDetail()

    const button = await screen.findByRole('button', { name: '积分不足' })
    expect(button).toBeDisabled()
    expect(screen.getByText('380 分')).toBeInTheDocument()
  })

  it('兑换时撞上库存不足（409 OUT_OF_STOCK）展示后端文案且不跳转', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', () =>
        failJson(409, 'OUT_OF_STOCK', '商品库存不足，兑换失败'),
      ),
    )
    renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(await screen.findByText('商品库存不足，兑换失败')).toBeInTheDocument()
    expect(screen.queryByTestId('route-stub')).not.toBeInTheDocument()
  })

  it('兑换成功后跳到兑换记录页', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', () =>
        okJson(
          {
            id: 7,
            userId: 1005,
            orderNo: 'GT202611240001',
            productId: 1,
            skuId: 1,
            productName: '环保帆布袋',
            skuName: '默认',
            pointsSpent: 100,
            quantity: 1,
            receiver: '本人',
            status: 'PAID',
            redeemCode: 'RC-0007',
            createdAt: '2026-11-24T02:00:00Z',
            updatedAt: '2026-11-24T02:00:00Z',
          },
          { status: 201, message: 'created' },
        ),
      ),
    )
    renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
  })
  it('非安全上下文没有 randomUUID 时仍能弹窗并带上 32 位幂等键', async () => {
    const keys: string[] = []
    // 局域网 http 演示：crypto 在，但 randomUUID 是 [SecureContext] 专属
    vi.stubGlobal('crypto', {
      getRandomValues: (bytes: Uint8Array) => {
        for (let i = 0; i < bytes.length; i += 1) bytes[i] = i
        return bytes
      },
    } as unknown as Crypto)
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', ({ request }) => {
        keys.push(request.headers.get('Idempotency-Key') ?? '')
        return okJson(makeOrder())
      }),
    )
    renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
    expect(keys).toHaveLength(1)
    expect(keys[0]).toMatch(/^[0-9a-f]{32}$/)
  })

  it('点「再想想」不会重新生成幂等键', async () => {
    const keys: string[] = []
    const randomUUID = vi.fn(() => 'ABCDEF01-2345-6789-ABCD-EF0123456789')
    vi.stubGlobal('crypto', {
      randomUUID,
      getRandomValues: (bytes: Uint8Array) => bytes,
    } as unknown as Crypto)
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', ({ request }) => {
        keys.push(request.headers.get('Idempotency-Key') ?? '')
        return okJson(makeOrder())
      }),
    )
    renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '再想想' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
    // 第一次点「确认兑换」就分配了键；「再想想」只是关弹窗，不该再分配一个
    expect(randomUUID).toHaveBeenCalledTimes(1)
    expect(keys).toHaveLength(1)
    expect(keys[0]).toBe('abcdef0123456789abcdef0123456789')
  })

  it('兑换成功后重新发起会换一个新幂等键', async () => {
    const keys: string[] = []
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', ({ request }) => {
        keys.push(request.headers.get('Idempotency-Key') ?? '')
        return okJson(makeOrder())
      }),
    )
    const first = renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))
    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()

    // 重新挂载等价于用户再进一次详情页：上一笔已成功，不该复用旧键
    first.unmount()
    renderDetail()
    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(keys).toHaveLength(2)
    expect(keys[1]).not.toBe(keys[0])
  })

  it('兑换失败后重试复用同一个幂等键', async () => {
    const keys: string[] = []
    let calls = 0
    server.use(
      http.get('*/api/v1/products/1', () => okJson(makeProduct())),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.post('*/api/v1/orders', ({ request }) => {
        keys.push(request.headers.get('Idempotency-Key') ?? '')
        calls += 1
        if (calls === 1) {
          return failJson(500, 'INTERNAL', '服务器内部错误')
        }
        return okJson(
          {
            id: 7,
            userId: 1005,
            orderNo: 'GT202611240001',
            productId: 1,
            skuId: 1,
            productName: '环保帆布袋',
            skuName: '默认',
            pointsSpent: 100,
            quantity: 1,
            receiver: '本人',
            status: 'PAID',
            redeemCode: 'RC-0007',
            createdAt: '2026-11-24T02:00:00Z',
            updatedAt: '2026-11-24T02:00:00Z',
          },
          { status: 201, message: 'created' },
        )
      }),
    )
    renderDetail()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))
    expect(await screen.findByText('服务器内部错误')).toBeInTheDocument()

    await user.click(await screen.findByRole('button', { name: '确认兑换' }))
    await user.click(await screen.findByRole('button', { name: '确认兑换并生成核销码' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
    expect(keys).toHaveLength(2)
    expect(keys[0]).not.toBe('')
    expect(keys[1]).toBe(keys[0])
  })

  it('商品详情 500 显示加载失败与重试，而不是「商品不存在」', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => failJson(500, 'INTERNAL', '数据库连接失败')),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
    )
    renderDetail()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('数据库连接失败')
    expect(screen.queryByText('商品不存在或已下架。')).not.toBeInTheDocument()
  })

  it('商品 404 才显示「商品不存在或已下架」', async () => {
    server.use(
      http.get('*/api/v1/products/1', () => failJson(404, 'NOT_FOUND', '商品不存在')),
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
    )
    renderDetail()

    expect(await screen.findByText('商品不存在或已下架。')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
