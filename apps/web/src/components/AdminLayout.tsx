import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { getSession, logout } from '@/services/auth'

const NAV_ITEMS = [
  { to: '/admin/products', label: '商品管理', end: true },
  { to: '/admin/products/new', label: '新建商品', end: true },
  { to: '/admin/orders', label: '订单管理', end: true },
]

/**
 * 运营后台外壳：与消费者端 AppLayout 分开，导航只保留后台三块功能。
 * 管理员身份由路由上的 AdminGate（探测 /admin/products 的 200/403）把住，
 * 普通市民根本进不到这个外壳，不会看到后台导航与表单。
 */
export function AdminLayout() {
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
          <span className="app__logo" aria-hidden="true">🛠️</span>
          <div>
            <h1 className="app__title">运营后台</h1>
            <p className="app__subtitle">绿行积分 · 商品与兑换订单</p>
          </div>
        </div>
        {session && (
          <span className="badge badge--ok">
            {session.nickname || session.userId}
            <button type="button" onClick={onLogout} className="btn btn--ghost btn--inline">
              退出
            </button>
          </span>
        )}
      </header>

      <nav className="app__nav" aria-label="后台导航">
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
    </div>
  )
}
