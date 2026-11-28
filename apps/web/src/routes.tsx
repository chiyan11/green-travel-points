import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/AppLayout'
import { AdminLayout } from '@/components/AdminLayout'
import { RequireAuth } from '@/components/RequireAuth'
import { AdminGate } from '@/components/AdminGate'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { TripsPage } from '@/pages/TripsPage'
import { PointsPage } from '@/pages/PointsPage'
import { MallPage } from '@/pages/MallPage'
import { ProductDetailPage } from '@/pages/ProductDetailPage'
import { OrdersPage } from '@/pages/OrdersPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { AdminProductsPage } from '@/pages/admin/AdminProductsPage'
import { AdminProductNewPage } from '@/pages/admin/AdminProductNewPage'
import { AdminOrdersPage } from '@/pages/admin/AdminOrdersPage'

/**
 * 路由：/login 公开；消费者端页面经 RequireAuth + AppLayout；
 * 运营后台（M8）挂在 /admin：RequireAuth 保证已登录，AdminGate 再探测一次后端
 * （200/403）确认是 ADMIN，普通市民只看到无权限面板，不渲染后台外壳与导航。
 *
 * 消费者端 8 个页面均已接入真实后端（请求只在 src/services/* 里发），
 * wire 类型由 docs/api/openapi.yaml 生成到 src/types/api.ts，页面视图模型见 src/types/domain.ts。
 */
export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: 'trips', element: <TripsPage /> },
      { path: 'points', element: <PointsPage /> },
      { path: 'mall', element: <MallPage /> },
      { path: 'mall/:id', element: <ProductDetailPage /> },
      { path: 'orders', element: <OrdersPage /> },
      { path: 'profile', element: <ProfilePage /> },
    ],
  },
  {
    path: '/admin',
    element: (
      <RequireAuth>
        <AdminGate>
          <AdminLayout />
        </AdminGate>
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="/admin/products" replace /> },
      { path: 'products', element: <AdminProductsPage /> },
      { path: 'products/new', element: <AdminProductNewPage /> },
      { path: 'orders', element: <AdminOrdersPage /> },
    ],
  },
])
