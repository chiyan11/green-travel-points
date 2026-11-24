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

/** 状态文字同时出现在筛选按钮和表格徽标里，断言表格内的那一份。 */
function table(): HTMLElement {
  return screen.getByRole('table')
}

describe('OrdersPage', () => {
  it('取消待支付订单后列表刷新为已取消', async () => {
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

    expect(await screen.findByRole('table')).toBeInTheDocument()
    expect(within(table()).getByText('待支付')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '取消并返还' }))

    await waitFor(() => expect(cancelCalls).toBe(1))
    await waitFor(() => expect(within(table()).getByText('已取消')).toBeInTheDocument())
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
    expect(within(table()).getByText('待支付')).toBeInTheDocument()
  })

  it('筛选已发货时显示对应空态', async () => {
    server.use(http.get('*/api/v1/orders', () => okJson(pageOf([makeOrder()]))))
    renderPage(<OrdersPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: '已发货' }))

    expect(await screen.findByText('该状态下暂无订单。')).toBeInTheDocument()
  })
})
