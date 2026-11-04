import { useQuery } from '@tanstack/react-query'
import { getRecentCarbonSeries, getSummary, listTxns } from '@/services/points'
import { POINT_TXN_TYPE_META } from '@/lib/orderStatus'
import { CarbonTrendChart } from '@/components/CarbonTrendChart'
import { formatDateTime, formatGrams, formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

export function PointsPage(): ReactNode {
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => Promise.resolve(getSummary()),
  })
  const { data: txns = [] } = useQuery({
    queryKey: ['txns'],
    queryFn: () => Promise.resolve(listTxns()),
  })
  const { data: trend = [] } = useQuery({
    queryKey: ['carbon-trend-7d'],
    queryFn: () => Promise.resolve(getRecentCarbonSeries()),
  })

  return (
    <>
      <section className="hero">
        <h2>我的积分</h2>
        <p>余额与流水只通过账本变更；当日发放受 500 分上限约束。</p>
      </section>

      <div className="grid">
        <div className="card">
          <p className="muted">当前余额</p>
          <p className="metric">{formatPoints(summary?.balance ?? 0)} 分</p>
        </div>
        <div className="card">
          <p className="muted">累计碳减排</p>
          <p className="metric">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
        </div>
        <div className="card">
          <p className="muted">本月获得</p>
          <p className="metric">{formatPoints(summary?.earnedThisMonth ?? 0)} 分</p>
        </div>
        <div className="card">
          <p className="muted">今日已发 / 上限</p>
          <p className="metric">
            {summary?.earnedToday ?? 0} / {summary?.dailyCap ?? 500}
          </p>
        </div>
      </div>

      <section className="card">
        <h3>近 7 日碳减排</h3>
        <CarbonTrendChart data={trend} />
      </section>

      <section className="card">
        <h3>积分流水</h3>
        {txns.length === 0 ? (
          <p className="empty-state">暂无流水。</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>类型</th>
                <th>变动</th>
                <th>变动后余额</th>
              </tr>
            </thead>
            <tbody>
              {txns.map((t) => {
                const meta = POINT_TXN_TYPE_META[t.type]
                const sign = t.amount >= 0 ? '+' : ''
                return (
                  <tr key={t.id}>
                    <td>{formatDateTime(t.occurredAt)}</td>
                    <td>{meta.label}</td>
                    <td className={t.amount >= 0 ? 'ok' : 'danger'}>
                      {sign}
                      {t.amount}
                    </td>
                    <td>{formatPoints(t.balanceAfter)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </section>
    </>
  )
}
