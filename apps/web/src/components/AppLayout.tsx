import { NavLink, Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchPing } from '@/lib/http'

const NAV_ITEMS = [
  { to: '/', label: '首页', end: true },
  { to: '/trips', label: '行程上报', end: false },
  { to: '/points', label: '我的积分', end: false },
  { to: '/mall', label: '积分商城', end: false },
  { to: '/orders', label: '兑换记录', end: false },
  { to: '/profile', label: '个人中心', end: false },
]

/** 后端连通性徽标：Sprint 0 用来证明前后端已打通。 */
function BackendBadge() {
  const { data, isPending, isError } = useQuery({
    queryKey: ['ping'],
    queryFn: fetchPing,
  })

  if (isPending) return <span className="badge badge--muted">后端检测中…</span>
  if (isError) return <span className="badge badge--danger">后端未连接</span>
  return <span className="badge badge--ok">后端正常 · {data.env}</span>
}

export function AppLayout() {
  return (
    <div className="app">
      <header className="app__header">
        <div className="app__brand">
          <span className="app__logo" aria-hidden="true">🌿</span>
          <div>
            <h1 className="app__title">绿行积分</h1>
            <p className="app__subtitle">城市绿色出行碳积分平台</p>
          </div>
        </div>
        <BackendBadge />
      </header>

      <nav className="app__nav" aria-label="主导航">
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) => (isActive ? 'nav-link nav-link--active' : 'nav-link')}
          >
            {item.label}
          </NavLink>
        ))}
      </nav>

      <main className="app__main">
        <Outlet />
      </main>

      <footer className="app__footer">
        软件工程实验 · 第 14 题 碳中和绿色出行积分系统 · Sprint 0 骨架
      </footer>
    </div>
  )
}
