import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/AppLayout'
import { AdminLayout } from '@/components/AdminLayout'
import { RequireAuth } from '@/components/RequireAuth'
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
 * 运营后台（M8）挂在 /admin，独立布局，管理员身份由后端 RequireAdmin 判定。
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
        <AdminLayout />
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
