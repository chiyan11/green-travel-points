import { useQuery } from '@tanstack/react-query'
import { Card } from '@astryxdesign/core/Card'
import { getCarbonSeries, getSummary, listTxns } from '@/services/points'
import { POINT_TXN_TYPE_META } from '@/lib/orderStatus'
import { CarbonTrendChart } from '@/components/CarbonTrendChart'
import { formatDateTime, formatGrams, formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

const STATS = [
  { icon: '💰', label: '当前余额', hint: '可兑换商城商品', accent: true },
  { icon: '🌿', label: '累计碳减排', hint: '全部行程累计', accent: false },
  { icon: '📅', label: '本月获得', hint: '按自然月统计', accent: false },
  { icon: '⏱️', label: '今日已发 / 上限', hint: '超出上限的部分不计积分', accent: false },
] as const

export function PointsPage(): ReactNode {
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => getSummary(),
  })
  const { data: txns = [] } = useQuery({
    queryKey: ['txns'],
    queryFn: () => listTxns(),
  })
  const { data: trend = [] } = useQuery({
    queryKey: ['carbon-trend-7d'],
    queryFn: () => getCarbonSeries(7),
  })

  const values: Record<string, string> = {
    当前余额: `${formatPoints(summary?.balance ?? 0)} 分`,
    累计碳减排: formatGrams(summary?.totalCarbonSavedGrams ?? 0),
    本月获得: `${formatPoints(summary?.earnedThisMonth ?? 0)} 分`,
    '今日已发 / 上限': `${summary?.earnedToday ?? 0} / ${summary?.dailyCap ?? 500}`,
  }

  return (
    <>
      <section className="hero">
        <h2>我的积分</h2>
        <p>余额与流水只通过账本变更；当日发放受 500 分上限约束。</p>
      </section>

      <div className="grid grid--stats">
        {STATS.map((s) => (
          <Card key={s.label} padding={4} elevation="low">
            <div className={s.accent ? 'stat-card stat-card--accent' : 'stat-card'}>
              <span aria-hidden="true">{s.icon}</span>
              <p className="stat-card__label">{s.label}</p>
              <p className="stat-card__value">{values[s.label]}</p>
              <p className="stat-card__hint">{s.hint}</p>
            </div>
          </Card>
        ))}
      </div>

      <Card padding={5} elevation="low">
        <h3 style={{ marginTop: 0 }}>近 7 日碳减排</h3>
        <CarbonTrendChart data={trend} />
      </Card>

      <Card padding={5} elevation="low">
        <h3 style={{ marginTop: 0 }}>积分流水</h3>
        {txns.length === 0 ? (
          <p className="empty-state">暂无流水。</p>
        ) : (
          <div className="table-wrap">
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
          </div>
        )}
      </Card>
    </>
  )
}
