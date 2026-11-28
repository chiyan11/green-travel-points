import { screen } from '@testing-library/react'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { AdminGate } from '@/components/AdminGate'
import { AdminLayout } from '@/components/AdminLayout'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'

/** 用真实的后台外壳包一层门，断言“无权限时连导航都不渲染”。 */
function renderAdmin() {
  return renderPage(
    <AdminGate>
      <AdminLayout />
    </AdminGate>,
    { route: '/admin/products', path: '/admin/products' },
  )
}

describe('AdminGate 角色门', () => {
  it('探测接口返回 403：只显示无权限面板，不渲染后台导航', async () => {
    server.use(
      http.get('*/api/v1/admin/products', () =>
        failJson(403, 'FORBIDDEN', '当前账号没有管理员权限'),
      ),
    )

    renderAdmin()

    expect(await screen.findByTestId('admin-denied')).toBeInTheDocument()
    expect(screen.getByText('当前账号没有管理员权限')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '返回首页' })).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: '后台导航' })).not.toBeInTheDocument()
    expect(screen.queryByText('新建商品')).not.toBeInTheDocument()
  })

  it('探测接口返回 200：管理员正常进入，后台导航齐全', async () => {
    server.use(http.get('*/api/v1/admin/products', () => okJson(pageOf([]))))

    renderAdmin()

    expect(await screen.findByRole('navigation', { name: '后台导航' })).toBeInTheDocument()
    expect(screen.queryByTestId('admin-denied')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品管理' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '订单管理' })).toBeInTheDocument()
  })
})
