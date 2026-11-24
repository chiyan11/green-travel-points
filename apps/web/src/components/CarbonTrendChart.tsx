/**
 * 近 7 日碳减排柱状图。
 *
 * 数据来自 GET /api/v1/points/daily-carbon（services/points#getCarbonSeries）。
 * 用 CSS 柱状而不是折线：这条数据绝大多数日子是 0，折线贴底再突然飙起，观感比柱状差得多。
 */
import type { ReactNode } from 'react'
import type { DailyCarbonPoint } from '@/services/points'

/** 数据太稀疏时不画图，避免「六天一条平线」的错觉。 */
const MIN_ACTIVE_DAYS = 3

function shortGrams(grams: number): string {
  if (grams >= 1000) return `${(grams / 1000).toFixed(1)}kg`
  return `${Math.round(grams)}g`
}

export function CarbonTrendChart({ data }: { data: DailyCarbonPoint[] }): ReactNode {
  const activeDays = data.filter((d) => d.grams > 0).length

  if (data.length === 0 || activeDays < MIN_ACTIVE_DAYS) {
    return (
      <div className="empty">
        <p className="empty__title">还看不出趋势</p>
        <p className="empty__note">攒够 3 天有出行记录的数据后，这里会显示近 7 日碳减排。</p>
      </div>
    )
  }

  const max = Math.max(...data.map((d) => d.grams), 1)
  const todayIndex = data.length - 1

  return (
    <div className="bars" role="img" aria-label="近 7 日碳减排柱状图">
      {data.map((d, i) => {
        const isToday = i === todayIndex
        const height = Math.max(3, Math.round((d.grams / max) * 100))
        return (
          <div className={isToday ? 'bar bar--today' : 'bar'} key={d.date}>
            <span className={isToday ? 'bar__value bar__value--amber' : 'bar__value'}>
              {d.grams > 0 ? shortGrams(d.grams) : ''}
            </span>
            <span className="bar__track">
              <span className="bar__fill" style={{ height: `${height}%` }} title={`${d.date} ${shortGrams(d.grams)}`} />
            </span>
            <span className="bar__label">{d.date}</span>
          </div>
        )
      })}
    </div>
  )
}
