import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { MallPage } from '@/pages/MallPage'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { PointsSummary, Product } from '@/types/api'

const summary: PointsSummary = {
  balance: 1000,
  frozen: 0,
  available: 1000,
  dailyCap: 500,
  earnedThisMonth: 120,
  earnedToday: 40,
  totalCarbonSavedGrams: 25600,
}

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

describe('MallPage', () => {
  it('商品列表为空时给出空态提示，而不是一片空白', async () => {
    server.use(
      http.get('*/api/v1/products', () => okJson(pageOf([]))),
      http.get('*/api/v1/points/summary', () => okJson(summary)),
    )
    renderPage(<MallPage />)

    expect(await screen.findByText('暂无在售商品，等上架后再来看看。')).toBeInTheDocument()
    expect(screen.getByText('共 0 件在售商品')).toBeInTheDocument()
    expect(await screen.findByText('我的余额 1,000 分')).toBeInTheDocument()
  })

  it('有商品时渲染价格与库存，并链到详情页', async () => {
    server.use(
      http.get('*/api/v1/products', () => okJson(pageOf([makeProduct()]))),
      http.get('*/api/v1/points/summary', () => okJson(summary)),
    )
    renderPage(<MallPage />)

    expect(await screen.findByRole('heading', { name: '环保帆布袋' })).toBeInTheDocument()
    expect(screen.getByText('共 1 件在售商品')).toBeInTheDocument()
    expect(screen.getByText('100 分')).toBeInTheDocument()
    expect(screen.getByText('库存 49')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '查看详情' })).toHaveAttribute('href', '/mall/1')
  })

  it('商品列表加载失败时给出失败态与重试，而不是空态', async () => {
    let calls = 0
    server.use(
      http.get('*/api/v1/products', () => {
        calls += 1
        if (calls === 1) return failJson(500, 'INTERNAL', '数据库连接失败')
        return okJson(pageOf([makeProduct()]))
      }),
      http.get('*/api/v1/points/summary', () => okJson(summary)),
    )
    renderPage(<MallPage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('数据库连接失败')
    // 空态文案不能出现：那是「真的没商品」，不是「没取到」
    expect(screen.queryByText('暂无在售商品，等上架后再来看看。')).not.toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: '重试' }))
    expect(await screen.findByText('环保帆布袋')).toBeInTheDocument()
  })
})
