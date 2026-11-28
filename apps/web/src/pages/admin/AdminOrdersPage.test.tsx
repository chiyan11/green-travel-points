import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AdminOrdersPage } from '@/pages/admin/AdminOrdersPage'
import { renderPage } from '@/test/render'
import { okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { OrderStatus } from '@/types/api'

interface RawOrder {
  id: number
  userId: number
  orderNo: string
  productName: string
  skuName: string
  pointsSpent: number
  quantity: number
  receiver: string
  status: OrderStatus
  createdAt: string
}

function makeOrder(over: Partial<RawOrder> = {}): RawOrder {
  return {
    id: 1,
    userId: 1001,
    orderNo: 'GT202611280001',
    productName: '环保帆布袋',
    skuName: '默认',
    pointsSpent: 100,
    quantity: 1,
    receiver: '演示用户',
    status: 'PAID',
    createdAt: '2026-11-28T02:00:00Z',
    ...over,
  }
}

function rowOf(orderNo: string): HTMLElement {
  const cell = screen.getByText(orderNo)
  const row = cell.closest('tr')
  if (row === null) throw new Error(`找不到订单行 ${orderNo}`)
  return row
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('AdminOrdersPage 订单流转二次确认', () => {
  it('取消确认不发 POST；确认后只发一次，并带上目标状态与不可撤销提示', async () => {
    let status: OrderStatus = 'PAID'
    let shipCalls = 0
    server.use(
      http.get('*/api/v1/admin/orders', () => okJson(pageOf([makeOrder({ status })]))),
      http.post('*/api/v1/admin/orders/1/ship', () => {
        shipCalls += 1
        status = 'SHIPPED'
        return okJson(makeOrder({ status }))
      }),
    )
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    renderPage(<AdminOrdersPage />)
    const user = userEvent.setup()

    await screen.findByText('GT202611280001')
    await user.click(within(rowOf('GT202611280001')).getByRole('button', { name: '发货' }))

    expect(confirmSpy).toHaveBeenCalledTimes(1)
    const message = String(confirmSpy.mock.calls[0]?.[0] ?? '')
    expect(message).toContain('已发货')
    expect(message).toContain('不可撤销')
    expect(shipCalls).toBe(0)
    expect(within(rowOf('GT202611280001')).getByText('已确认 · 待发货')).toBeInTheDocument()

    confirmSpy.mockReturnValue(true)
    await user.click(within(rowOf('GT202611280001')).getByRole('button', { name: '发货' }))

    await waitFor(() => expect(shipCalls).toBe(1))
    await waitFor(() => expect(within(rowOf('GT202611280001')).getByText('已发货')).toBeInTheDocument())
    expect(confirmSpy).toHaveBeenCalledTimes(2)
  })

  it('A 行请求在途时只禁用 A 行，B 行仍可操作', async () => {
    const statuses: Record<number, OrderStatus> = { 1: 'PAID', 2: 'PAID' }
    server.use(
      http.get('*/api/v1/admin/orders', () =>
        okJson(
          pageOf([
            makeOrder({ id: 1, orderNo: 'GT-A', status: statuses[1] ?? 'PAID' }),
            makeOrder({ id: 2, orderNo: 'GT-B', status: statuses[2] ?? 'PAID' }),
          ]),
        ),
      ),
    )

    let release: (() => void) | undefined
    server.use(
      http.post('*/api/v1/admin/orders/1/ship', async () => {
        await new Promise<void>((resolve) => {
          release = resolve
        })
        statuses[1] = 'SHIPPED'
        return okJson(makeOrder({ id: 1, orderNo: 'GT-A', status: 'SHIPPED' }))
      }),
    )

    vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderPage(<AdminOrdersPage />)
    const user = userEvent.setup()

    await screen.findByText('GT-A')
    await user.click(within(rowOf('GT-A')).getByRole('button', { name: '发货' }))

    // 请求在途：A 行按钮变成「发货中…」并禁用，B 行按钮仍可用。
    const buttonA = await within(rowOf('GT-A')).findByRole('button', { name: '发货中…' })
    expect(buttonA).toBeDisabled()
    expect(within(rowOf('GT-B')).getByRole('button', { name: '发货' })).toBeEnabled()

    release?.()
    await waitFor(() => expect(within(rowOf('GT-A')).getByText('已发货')).toBeInTheDocument())
  })

  it('超过一页时给出分页，翻页请求 page=2 而不是把第二页截掉', async () => {
    const pages: string[] = []
    server.use(
      http.get('*/api/v1/admin/orders', ({ request }) => {
        const page = new URL(request.url).searchParams.get('page') ?? '1'
        pages.push(page)
        const items =
          page === '2'
            ? [makeOrder({ id: 2, orderNo: 'GT-P2' })]
            : [makeOrder({ id: 1, orderNo: 'GT-P1' })]
        return okJson({ items, page: Number(page), pageSize: 20, total: 21 })
      }),
    )

    renderPage(<AdminOrdersPage />)
    const user = userEvent.setup()

    expect(await screen.findByText(/共 21 单/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '下一页' }))

    expect(await screen.findByText('GT-P2')).toBeInTheDocument()
    expect(pages).toContain('2')
    expect(screen.getByText(/第 2 \/ 2 页/)).toBeInTheDocument()
  })
})
