import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { IconLeaf } from '@/components/icons'
import { getSummary } from '@/services/points'
import { getSession, logout } from '@/services/auth'
import { formatPoints } from '@/lib/format'

const NAV_ITEMS = [
  { to: '/', label: '首页', end: true },
  { to: '/trips', label: '行程上报', end: false },
  { to: '/points', label: '我的积分', end: false },
  { to: '/mall', label: '积分商城', end: false },
  { to: '/orders', label: '兑换记录', end: false },
  { to: '/profile', label: '个人中心', end: false },
]

/**
 * 登录后的应用外壳：顶栏（品牌 / 导航 / 余额 / 退出）、内容区、页脚。
 *
 * 余额常驻在顶栏：积分是这个产品的核心货币，用户不该为了看一眼余额而换页面。
 */
export function AppLayout() {
  const navigate = useNavigate()
  const session = getSession()

  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: getSummary,
    enabled: Boolean(session),
  })

  const onLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="app">
      <header className="app__header">
        <NavLink to="/" className="app__brand">
          <span className="app__logo">
            <IconLeaf size={22} />
          </span>
          <span>
            <span className="app__title">绿行积分</span>
            <span className="app__subtitle">城市绿色出行碳积分平台</span>
          </span>
        </NavLink>

        <nav className="app__nav" aria-label="主导航">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) => (isActive ? 'nav-link is-active' : 'nav-link')}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="app__aside">
          {summary && (
            <span className="balance-chip">可用 {formatPoints(summary.balance)} 分</span>
          )}
          {session && <span className="muted app__user">{session.nickname}</span>}
          {session && (
            <button type="button" className="btn btn--ghost" onClick={onLogout}>
              退出
            </button>
          )}
        </div>
      </header>

      <main className="app__main">
        <Outlet />
      </main>

      <footer className="app__footer">让每一次低碳出行，都被看见。</footer>
    </div>
  )
}
