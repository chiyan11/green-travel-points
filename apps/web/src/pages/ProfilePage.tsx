import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@astryxdesign/core/Badge'
import { Button } from '@astryxdesign/core/Button'
import { Card } from '@astryxdesign/core/Card'
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

      <Card padding={5} elevation="low">
        <div className="row-between">
          <div className="app__brand">
            <span className="app__logo" aria-hidden="true">
              👤
            </span>
            <div>
              <h3 style={{ margin: 0 }}>{session?.nickname ?? '未登录'}</h3>
              <p className="muted" style={{ margin: 0 }}>
                用户 ID：{session?.userId ?? '-'}
              </p>
            </div>
          </div>
          <Badge
            label={session ? '已登录 · 令牌 2 小时有效' : '未登录'}
            variant={session ? 'success' : 'neutral'}
          />
        </div>
        <div className="row-between" style={{ marginTop: 'var(--spacing-4)' }}>
          <p className="muted" style={{ margin: 0 }}>
            会话保存在浏览器 localStorage，退出后立即清除。
          </p>
          <Button label="退出登录" variant="secondary" size="sm" onClick={onLogout} />
        </div>
      </Card>

      <div className="grid grid--stats">
        <Card padding={4} elevation="low">
          <div className="stat-card stat-card--accent">
            <span aria-hidden="true">💰</span>
            <p className="stat-card__label">累计积分余额</p>
            <p className="stat-card__value">{formatPoints(summary?.balance ?? 0)} 分</p>
            <p className="stat-card__hint">可用于商城兑换</p>
          </div>
        </Card>
        <Card padding={4} elevation="low">
          <div className="stat-card">
            <span aria-hidden="true">🌿</span>
            <p className="stat-card__label">累计碳减排</p>
            <p className="stat-card__value">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
            <p className="stat-card__hint">基准：燃油私家车 192 gCO₂e/km</p>
          </div>
        </Card>
      </div>

      <Card padding={5} elevation="low">
        <div className="section-head">
          <h3>成就徽章</h3>
          <Badge label="待 Sprint 3 接入" variant="neutral" />
        </div>
        <p className="empty-state">
          连续绿色出行 7 天、累计减排 10kg 等里程碑徽章，由后端 Outbox 事件触发。
        </p>
      </Card>
    </>
  )
}
