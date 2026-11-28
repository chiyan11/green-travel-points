import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { OrdersPage } from '@/pages/OrdersPage'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { Order, OrderStatus } from '@/types/api'

function makeOrder(over: Partial<Order> = {}): Order {
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
    status: 'PENDING',
    redeemCode: '',
    createdAt: '2026-11-24T02:00:00Z',
    updatedAt: '2026-11-24T02:00:00Z',
    ...over,
  }
}

/** 状态文字同时出现在筛选按钮和订单卡片徽标里，断言卡片内的那一份。 */
function card(): HTMLElement {
  return screen.getByRole('article')
}

describe('OrdersPage', () => {
  it('取消待商户确认的订单后列表刷新为已取消', async () => {
    let status: OrderStatus = 'PENDING'
    let cancelCalls = 0
    server.use(
      http.get('*/api/v1/orders', () => okJson(pageOf([makeOrder({ status })]))),
      http.post('*/api/v1/orders/7/cancel', () => {
        cancelCalls += 1
        status = 'CANCELLED'
        return okJson(null)
      }),
    )
    renderPage(<OrdersPage />)
    const user = userEvent.setup()

    expect(await screen.findByRole('article')).toBeInTheDocument()
    expect(within(card()).getByText('待商户确认')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '取消并返还' }))

    await waitFor(() => expect(cancelCalls).toBe(1))
    await waitFor(() => expect(within(card()).getByText('已取消')).toBeInTheDocument())
    // 已取消的订单不再给出取消按钮。
    expect(screen.queryByRole('button', { name: '取消并返还' })).not.toBeInTheDocument()
  })

  it('取消失败（409 ILLEGAL_TRANSITION）时展示后端文案且保留订单', async () => {
    server.use(
      http.get('*/api/v1/orders', () => okJson(pageOf([makeOrder()]))),
      http.post('*/api/v1/orders/7/cancel', () =>
        failJson(409, 'ILLEGAL_TRANSITION', '订单已发货，不能取消'),
      ),
    )
    renderPage(<OrdersPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '取消并返还' }))

    expect(await screen.findByText('订单已发货，不能取消')).toBeInTheDocument()
    expect(within(card()).getByText('待商户确认')).toBeInTheDocument()
  })

  it('筛选已发货时把状态传给后端并显示对应空态', async () => {
    const seen: string[] = []
    server.use(
      http.get('*/api/v1/orders', ({ request }) => {
        const status = new URL(request.url).searchParams.get('status') ?? ''
        seen.push(status)
        // 后端按状态过滤：请求了 SHIPPED 就没有数据
        return okJson(status === 'SHIPPED' ? pageOf([]) : pageOf([makeOrder()]))
      }),
    )
    renderPage(<OrdersPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '已发货' }))

    expect(await screen.findByText('该状态下暂无订单。')).toBeInTheDocument()
    expect(seen).toContain('SHIPPED')
  })

  it('超过一页时给出上一页/下一页，翻页请求第二页', async () => {
    const pages: string[] = []
    server.use(
      http.get('*/api/v1/orders', ({ request }) => {
        const page = new URL(request.url).searchParams.get('page') ?? '1'
        pages.push(page)
        const items =
          page === '2'
            ? [makeOrder({ id: 9, productName: '咖啡馆兑换券' })]
            : [makeOrder({ id: 7 })]
        return okJson({ items, page: Number(page), pageSize: 20, total: 21 })
      }),
    )
    renderPage(<OrdersPage />)
    const user = userEvent.setup()

    expect(await screen.findByText(/共 21 条/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '下一页' }))

    expect(await screen.findByText('咖啡馆兑换券')).toBeInTheDocument()
    expect(pages).toContain('2')
    expect(screen.getByText(/第 2 \/ 2 页/)).toBeInTheDocument()
  })

  it('带 ?created= 进入时给出兑换成功提示并高亮该订单', async () => {
    server.use(http.get('*/api/v1/orders', () => okJson(pageOf([makeOrder({ id: 7 })]))))
    renderPage(<OrdersPage />, { route: '/orders?created=7', path: '/orders' })

    expect(await screen.findByRole('status')).toHaveTextContent('兑换成功')
    const highlighted = await screen.findByRole('article')
    expect(highlighted.className).toContain('order-card--highlight')
  })

  it('订单加载失败时给出失败态与重试，不显示「该状态下暂无订单」', async () => {
    let calls = 0
    server.use(
      http.get('*/api/v1/orders', () => {
        calls += 1
        if (calls === 1) return failJson(503, 'UNAVAILABLE', '服务暂时不可用')
        return okJson(pageOf([makeOrder()]))
      }),
    )
    renderPage(<OrdersPage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('服务暂时不可用')
    expect(screen.queryByText('该状态下暂无订单。')).not.toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: '重试' }))
    expect(await screen.findByText('环保帆布袋')).toBeInTheDocument()
  })
})
