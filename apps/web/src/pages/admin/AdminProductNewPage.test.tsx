import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { AdminProductNewPage } from '@/pages/admin/AdminProductNewPage'
import { AdminProductsPage } from '@/pages/admin/AdminProductsPage'
import { renderPage } from '@/test/render'
import { okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'

interface RawProduct {
  id: number
  merchantId: number
  name: string
  description: string
  category: string
  status: string
  pointsPrice: number
  stock: number
  limitPerUser: number
  skus: Array<{ id: number; name: string; pointsPrice: number; stock: number }>
}

function makeProduct(over: Partial<RawProduct> = {}): RawProduct {
  return {
    id: 1,
    merchantId: 1001,
    name: '环保帆布袋',
    description: '',
    category: '生活',
    status: 'ONLINE',
    pointsPrice: 100,
    stock: 10,
    limitPerUser: 0,
    skus: [{ id: 1, name: '标准款', pointsPrice: 100, stock: 10 }],
    ...over,
  }
}

describe('AdminProductNewPage 创建后失效列表', () => {
  it('创建成功后列表立即重新拉取，新商品无需刷新即出现在表格里', async () => {
    let products: RawProduct[] = [makeProduct()]
    let listCalls = 0
    server.use(
      http.get('*/api/v1/admin/products', () => {
        listCalls += 1
        return okJson(pageOf(products))
      }),
      http.post('*/api/v1/admin/products', async ({ request }) => {
        const body = (await request.json()) as { name: string; category: string }
        const created = makeProduct({
          id: 99,
          name: body.name,
          category: body.category,
          status: 'DRAFT',
        })
        products = [created, ...products]
        return okJson(created, { status: 201 })
      }),
    )

    renderPage(
      <>
        <AdminProductsPage />
        <AdminProductNewPage />
      </>,
      { route: '/admin/products', path: '/admin/products' },
    )
    const user = userEvent.setup()

    // 首屏只拉了一次列表，新商品还不存在。
    expect(await screen.findByText('环保帆布袋')).toBeInTheDocument()
    expect(listCalls).toBe(1)

    await user.type(screen.getByLabelText('名称'), '竹纤维毛巾')
    await user.type(screen.getByLabelText('分类'), '生活')
    await user.click(screen.getByRole('button', { name: '创建商品' }))

    // 失效缓存 → 列表自动 refetch（顺序：GET 1 次首屏 + 创建后 1 次）。
    await waitFor(() => expect(listCalls).toBe(2))
    expect(within(screen.getByRole('table')).getByText('竹纤维毛巾')).toBeInTheDocument()
    expect(await screen.findByText('已创建：竹纤维毛巾')).toBeInTheDocument()
  })
})
