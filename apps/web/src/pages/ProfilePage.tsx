import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getSession, logoutMock } from '@/services/auth'
import { getSummary } from '@/services/points'
import { formatGrams, formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

export function ProfilePage(): ReactNode {
  const navigate = useNavigate()
  const session = getSession()
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => Promise.resolve(getSummary()),
  })

  const onLogout = () => {
    logoutMock()
    navigate('/login', { replace: true })
  }

  return (
    <>
      <section className="hero">
        <h2>个人中心</h2>
        <p>账号信息、碳减排成就与数据导出（原型仅展示）。</p>
      </section>
      <section className="card">
        <h3>账号</h3>
        <p>昵称：{session?.nickname ?? '未登录'}</p>
        <p className="muted">用户 ID：{session?.userId ?? '-'}</p>
        <button className="btn btn--ghost" onClick={onLogout}>
          退出登录（清除本地会话）
        </button>
      </section>
      <div className="grid">
        <div className="card">
          <p className="muted">累计积分余额</p>
          <p className="metric">{formatPoints(summary?.balance ?? 0)} 分</p>
        </div>
        <div className="card">
          <p className="muted">累计碳减排</p>
          <p className="metric">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
        </div>
      </div>
      <section className="card">
        <h3>成就徽章（原型占位）</h3>
        <p className="muted">连续绿色出行 7 天 / 累计减排 10kg 等里程碑徽章，由后端 Outbox 事件触发，待 Sprint 3 接入。</p>
      </section>
    </>
  )
}
