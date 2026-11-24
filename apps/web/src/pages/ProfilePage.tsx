import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { IconLeaf, IconUser } from '@/components/icons'
import { getSession, logout } from '@/services/auth'
import { getSummary } from '@/services/points'
import { formatGrams, formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

/** 个人中心：账号信息与累计数据，不放开发态说明。 */
export function ProfilePage(): ReactNode {
  const navigate = useNavigate()
  const session = getSession()
  const { data: summary } = useQuery({ queryKey: ['summary'], queryFn: () => getSummary() })

  const onLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="page">
      <div className="page__head">
        <div>
          <h1 className="page-title">个人中心</h1>
          <p className="page-sub">账号信息与累计碳减排成就。</p>
        </div>
      </div>

      <section className="card">
        <div className="card__head">
          <div className="app__brand">
            <span className="app__logo">
              <IconUser size={22} />
            </span>
            <span>
              <span className="app__title">{session?.nickname ?? '未登录'}</span>
              <span className="app__subtitle">用户 ID：{session?.userId ?? '-'}</span>
            </span>
          </div>
          <button type="button" className="btn" onClick={onLogout}>
            退出登录
          </button>
        </div>
        <p className="muted">登录有效期 2 小时，到期后需要重新登录。</p>
      </section>

      <div className="stat-row">
        <section className="stat stat--main">
          <p className="stat__label">可用积分</p>
          <p className="stat__value">
            {formatPoints(summary?.balance ?? 0)}
            <span className="stat__unit">分</span>
          </p>
          <p className="stat__note">可用于商城兑换</p>
        </section>
        <section className="stat">
          <p className="stat__label">累计碳减排</p>
          <p className="stat__value">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
          <p className="stat__note">基准：燃油私家车 192 gCO₂e/km</p>
        </section>
        <section className="stat">
          <p className="stat__label">本月获得</p>
          <p className="stat__value">{formatPoints(summary?.earnedThisMonth ?? 0)} 分</p>
          <p className="stat__note">按自然月统计</p>
        </section>
      </div>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">成就徽章</h2>
          <span className="badge badge--muted">即将开放</span>
        </div>
        <div className="empty">
          <span className="empty__art">
            <IconLeaf size={44} />
          </span>
          <p className="empty__title">还没有点亮徽章</p>
          <p className="empty__note">连续绿色出行、累计减排达到里程碑后，这里会出现对应徽章。</p>
        </div>
      </section>
    </div>
  )
}
