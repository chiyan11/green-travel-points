import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@astryxdesign/core/Badge'
import { Button } from '@astryxdesign/core/Button'
import { fetchPing } from '@/lib/http'
import { getSession, logout } from '@/services/auth'

const NAV_ITEMS = [
  { to: '/', label: '首页', end: true },
  { to: '/trips', label: '行程上报', end: false },
  { to: '/points', label: '我的积分', end: false },
  { to: '/mall', label: '积分商城', end: false },
  { to: '/orders', label: '兑换记录', end: false },
  { to: '/profile', label: '个人中心', end: false },
]

/** 后端连通性徽标：证明前后端已打通。 */
function BackendBadge() {
  const { data, isPending, isError } = useQuery({
    queryKey: ['ping'],
    queryFn: fetchPing,
  })

  if (isPending) return <Badge label="后端检测中…" variant="neutral" />
  if (isError) return <Badge label="后端未连接" variant="error" />
  return <Badge label={`后端正常 · ${data.env}`} variant="success" />
}

export function AppLayout() {
  const navigate = useNavigate()
  const session = getSession()

  const onLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="app">
      <header className="app__header">
        <div className="app__brand">
          <span className="app__logo" aria-hidden="true">
            🌿
          </span>
          <div>
            <h1 className="app__title">绿行积分</h1>
            <p className="app__subtitle">城市绿色出行碳积分平台</p>
          </div>
        </div>
        <div className="app__header-actions">
          <BackendBadge />
          {session && (
            <>
              <Badge label={session.nickname} variant="neutral" />
              <Button label="退出" variant="ghost" size="sm" onClick={onLogout} />
            </>
          )}
        </div>
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
        软件工程实验 · 第 14 题 碳中和绿色出行积分系统 · 前端已对接后端接口
      </footer>
    </div>
  )
}
