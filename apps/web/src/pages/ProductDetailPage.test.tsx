import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
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

function renderDetail() {
  return renderPage(<ProductDetailPage />, { route: '/mall/1', path: '/mall/:id' })
}

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
    expect(screen.getByText('（还差 380 分）')).toBeInTheDocument()
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

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
  })
})
