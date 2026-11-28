import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { AdminProductsPage } from '@/pages/admin/AdminProductsPage'
import { renderPage } from '@/test/render'
import { okJson } from '@/test/http'
import { server } from '@/test/server'

interface RawProduct {
  id: number
  name: string
  category: string
  status: string
  pointsPrice: number
  stock: number
}

function makeProduct(over: Partial<RawProduct> = {}): RawProduct {
  return {
    id: 1,
    name: '环保帆布袋',
    category: '文创',
    status: 'ONLINE',
    pointsPrice: 100,
    stock: 40,
    ...over,
  }
}

describe('AdminProductsPage', () => {
  it('超过一页时给出分页，翻页请求 page=2 并带上关键词与状态', async () => {
    const calls: Array<{ page: string; keyword: string; status: string }> = []
    server.use(
      http.get('*/api/v1/admin/products', ({ request }) => {
        const url = new URL(request.url)
        const page = url.searchParams.get('page') ?? '1'
        calls.push({
          page,
          keyword: url.searchParams.get('keyword') ?? '',
          status: url.searchParams.get('status') ?? '',
        })
        const items =
          page === '2'
            ? [makeProduct({ id: 2, name: '咖啡馆兑换券' })]
            : [makeProduct({ id: 1 })]
        return okJson({ items, page: Number(page), pageSize: 20, total: 21 })
      }),
    )

    renderPage(<AdminProductsPage />)
    const user = userEvent.setup()

    expect(await screen.findByText(/共 21 件/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '下一页' }))

    expect(await screen.findByText('咖啡馆兑换券')).toBeInTheDocument()
    expect(calls.some((c) => c.page === '2')).toBe(true)
    expect(screen.getByText(/第 2 \/ 2 页/)).toBeInTheDocument()
  })

  it('搜索后回到第 1 页，不会停在旧页码上', async () => {
    const pages: string[] = []
    server.use(
      http.get('*/api/v1/admin/products', ({ request }) => {
        const url = new URL(request.url)
        pages.push(url.searchParams.get('page') ?? '1')
        return okJson({ items: [makeProduct()], page: 1, pageSize: 20, total: 21 })
      }),
    )

    renderPage(<AdminProductsPage />)
    const user = userEvent.setup()

    await screen.findByText('环保帆布袋')
    await user.click(screen.getByRole('button', { name: '下一页' }))
    await user.type(screen.getByLabelText('搜索商品'), '帆布')
    await user.click(screen.getByRole('button', { name: '搜索' }))

    // 搜索把页码重置为 1：最后一次请求的 page 必须是 1
    expect(pages[pages.length - 1]).toBe('1')
  })
})
