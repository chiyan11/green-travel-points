import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  IconArrowRight,
  IconAward,
  IconGift,
  IconRoute,
  IconSpark,
} from '@/components/icons'
import { getSummary } from '@/services/points'
import { listTrips } from '@/services/trips'
import { formatDateTime, formatDistance, formatGrams, formatPoints } from '@/lib/format'
import { modeEmoji, modeLabel } from '@/lib/travelMode'

const STEPS = [
  { title: '上报行程', note: '选方式 + 填距离' },
  { title: '核算减排', note: '按出行方式折算' },
  { title: '发放积分', note: '100 g 碳减排 = 1 分' },
  { title: '商城兑换', note: '换绿色好物' },
]

const ENTRIES = [
  { to: '/trips', icon: IconRoute, title: '行程上报', note: '步行 / 骑行 / 地铁 / 公交' },
  { to: '/points', icon: IconAward, title: '我的积分', note: '余额、流水与近 7 日趋势' },
  { to: '/mall', icon: IconGift, title: '积分商城', note: '用碳积分换绿色好物' },
]

/** 首页：左边一句主张，右边一块实时数据，下面是流程与入口。 */
export function HomePage(): ReactNode {
  const { data: summary } = useQuery({ queryKey: ['summary'], queryFn: getSummary })
  const { data: trips } = useQuery({ queryKey: ['trips'], queryFn: listTrips })

  const savedGrams = summary?.totalCarbonSavedGrams ?? 0
  // 192 g/km 基准：把碳减排换算成「少开燃油车多少公里」，让数字有体感。
  const equivalentKm = Math.round(savedGrams / 192)
  const recent = (trips ?? []).slice(0, 3)

  return (
    <div className="page">
      <section className="hero">
        <div className="hero__copy">
          <h1 className="hero__title">把每一次低碳出行，算成看得见的积分</h1>
          <p className="hero__lede">
            步行、骑行、地铁、公交都算数。上报一次出行，系统按出行方式核算碳减排，攒下的积分可以在商城换绿色好物。
          </p>
          <div className="hero__cta">
            <Link className="btn btn--primary" to="/trips">
              上报一次出行
            </Link>
            <Link className="btn" to="/mall">
              去商城看看
            </Link>
          </div>
        </div>

        <div className="hero__stats">
          <div className="hero__stat">
            <p className="hero__stat-label">累计碳减排</p>
            <p className="hero__stat-value">
              {formatGrams(savedGrams).split(' ')[0]}
              <span className="hero__stat-unit">{formatGrams(savedGrams).split(' ')[1]}</span>
            </p>
            <p className="hero__stat-note">相当于少开燃油车约 {formatPoints(equivalentKm)} 公里</p>
          </div>
          <div className="hero__stat">
            <p className="hero__stat-label">可用积分</p>
            <p className="hero__stat-value hero__stat-value--amber">
              {summary ? formatPoints(summary.balance) : '—'}
              <span className="hero__stat-unit"> 分</span>
            </p>
            <p className="hero__stat-note">本月已获得 {formatPoints(summary?.earnedThisMonth ?? 0)} 分</p>
          </div>
        </div>
      </section>

      <ol className="steps">
        {STEPS.map((step, i) => (
          <li className="step" key={step.title}>
            <p className="step__title">
              {i + 1}. {step.title}
            </p>
            <p className="step__note">{step.note}</p>
          </li>
        ))}
      </ol>

      <section className="grid grid--entries">
        {ENTRIES.map(({ to, icon: Icon, title, note }) => (
          <Link className="entry" to={to} key={to}>
            <span className="entry__icon">
              <Icon size={20} />
            </span>
            <span>
              <span className="entry__title">{title}</span>
              <span className="entry__note">{note}</span>
            </span>
            <span className="entry__arrow">
              <IconArrowRight size={18} />
            </span>
          </Link>
        ))}
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">最近行程</h2>
          <Link className="btn btn--ghost btn--inline" to="/trips">
            全部行程
          </Link>
        </div>

        {recent.length === 0 ? (
          <div className="empty">
            <span className="empty__art">
              <IconSpark size={32} />
            </span>
            <p className="empty__title">还没有行程记录</p>
            <p className="empty__note">上报第一程，就能看到碳减排和积分。</p>
            <Link className="btn btn--primary" to="/trips">
              上报第一程
            </Link>
          </div>
        ) : (
          <div className="list">
            {recent.map((trip) => (
              <div className="list__row" key={trip.id}>
                <span className="list__icon">
                  <IconRoute size={16} />
                </span>
                <div className="list__body">
                  <p className="list__title">
                    {modeEmoji(trip.mode)} {modeLabel(trip.mode)} · {formatDistance(trip.distanceKm)}
                  </p>
                  <p className="list__meta">{formatDateTime(trip.startedAt)}</p>
                </div>
                <span className="list__value list__value--amber">
                  +{formatPoints(trip.grantedPoints)}
                </span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
