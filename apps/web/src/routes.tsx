import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/components/AppLayout'
import { RequireAuth } from '@/components/RequireAuth'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { TripsPage } from '@/pages/TripsPage'
import { PointsPage } from '@/pages/PointsPage'
import { MallPage } from '@/pages/MallPage'
import { ProductDetailPage } from '@/pages/ProductDetailPage'
import { OrdersPage } from '@/pages/OrdersPage'
import { ProfilePage } from '@/pages/ProfilePage'

/**
 * 路由：/login 公开；其余页面经 RequireAuth 守卫后进入布局。
 * 8 个页面均已接入真实后端（请求只在 src/services/* 里发）；类型契约仍是手写过渡版，
 * 见 src/types/domain.ts 头注释。
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
])
