import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getSession, logout } from '@/services/auth'
import { getSummary } from '@/services/points'
import { formatGrams, formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

export function ProfilePage(): ReactNode {
  const navigate = useNavigate()
  const session = getSession()
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => getSummary(),
  })

  const onLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <>
      <section className="hero">
        <h2>个人中心</h2>
        <p>账号信息、碳减排成就与数据导出（原型仅展示）。</p>
      </section>

      <section className="card">
        <div className="row-between">
          <div className="app__brand">
            <span className="app__logo" aria-hidden="true">👤</span>
            <div>
              <h3 style={{ margin: 0 }}>{session?.nickname ?? '未登录'}</h3>
              <p className="muted" style={{ margin: 0 }}>
                用户 ID：{session?.userId ?? '-'}
              </p>
            </div>
          </div>
          <span className={session ? 'badge badge--ok' : 'badge badge--muted'}>
            {session ? '已登录 · 令牌 2 小时有效' : '未登录'}
          </span>
        </div>
        <div className="row-between" style={{ marginTop: 18 }}>
          <p className="muted" style={{ margin: 0 }}>
            会话保存在浏览器 localStorage，退出后立即清除。
          </p>
          <button className="btn btn--ghost" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </section>

      <div className="grid">
        <div className="stat-card stat-card--accent">
          <span className="stat-card__icon" aria-hidden="true">💰</span>
          <div>
            <p className="stat-label">累计积分余额</p>
            <p className="stat-value">{formatPoints(summary?.balance ?? 0)} 分</p>
            <p className="stat-sub">可用于商城兑换</p>
          </div>
        </div>
        <div className="stat-card">
          <span className="stat-card__icon" aria-hidden="true">🌿</span>
          <div>
            <p className="stat-label">累计碳减排</p>
            <p className="stat-value">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
            <p className="stat-sub">基准：燃油私家车 192 gCO₂e/km</p>
          </div>
        </div>
      </div>

      <section className="card">
        <div className="section-head">
          <h3>成就徽章</h3>
          <span className="badge badge--muted">待 Sprint 3 接入</span>
        </div>
        <p className="empty-state">
          连续绿色出行 7 天、累计减排 10kg 等里程碑徽章，由后端 Outbox 事件触发。
        </p>
      </section>
    </>
  )
}
