import { useState, type FormEvent, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { IconCheck, IconEmptyRoute, IconRoute } from '@/components/icons'
import { QueryErrorBlock } from '@/components/QueryError'
import { TRAVEL_MODES, type TravelMode, type TripRecord } from '@/types/domain'
import { validateDistance } from '@/lib/validators'
import { listTrips, submitTrip } from '@/services/trips'
import { formatDateTime, formatDistance, formatGrams, formatPoints } from '@/lib/format'
import { modeEmoji, modeLabel } from '@/lib/travelMode'

const QUICK_DISTANCES = [
  { label: '地铁 3 站 ≈ 5 km', value: '5' },
  { label: '骑行 3 km', value: '3' },
  { label: '步行 1.5 km', value: '1.5' },
]

/** 行程上报：选方式、填距离、看后端核算的结算单。 */
export function TripsPage(): ReactNode {
  const queryClient = useQueryClient()
  const {
    data: trips = [],
    isError: tripsError,
    error: tripsErrorObj,
    refetch: refetchTrips,
  } = useQuery({
    queryKey: ['trips'],
    queryFn: () => listTrips(),
  })

  const [mode, setMode] = useState<TravelMode>('subway')
  const [distanceInput, setDistanceInput] = useState('')
  const [startedAt, setStartedAt] = useState('')
  const [distanceError, setDistanceError] = useState('')
  const [last, setLast] = useState<TripRecord | null>(null)
  const [submitError, setSubmitError] = useState('')

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitError('')
    const check = validateDistance(distanceInput)
    if (!check.ok) {
      setDistanceError(check.message)
      return
    }
    setDistanceError('')
    try {
      const trip = await submitTrip({
        mode,
        distanceKm: check.value,
        startedAt: startedAt ? new Date(startedAt).toISOString() : new Date().toISOString(),
      })
      setLast(trip)
      setDistanceInput('')
      setStartedAt('')
      void queryClient.invalidateQueries({ queryKey: ['trips'] })
      void queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['txns'] })
      // 新的出行会改变当天的碳减排柱：不失效的话首页/积分页还显示提交前的图
      void queryClient.invalidateQueries({ queryKey: ['carbon-trend-7d'] })
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : '提交失败')
    }
  }

  return (
    <div className="page">
      <div className="page__head">
        <div>
          <h1 className="page-title">行程上报</h1>
          <p className="page-sub">选出行方式、填距离，碳减排和积分以后端核算为准。</p>
        </div>
        <span className="badge badge--muted">单日最多计 500 分</span>
      </div>

      <section className="card">
        <form className="form" onSubmit={onSubmit}>
          <div className="field">
            <span className="field__label">出行方式</span>
            <div className="mode-grid">
              {TRAVEL_MODES.map((m) => (
                <label key={m.code} className={mode === m.code ? 'mode is-active' : 'mode'}>
                  <input
                    type="radio"
                    name="mode"
                    value={m.code}
                    checked={mode === m.code}
                    onChange={() => setMode(m.code)}
                    className="visually-hidden"
                  />
                  <span className="mode__icon" aria-hidden="true">
                    {m.emoji}
                  </span>
                  <span className="mode__label">{m.label}</span>
                  <span className="mode__check" aria-hidden="true">
                    <IconCheck size={16} />
                  </span>
                </label>
              ))}
            </div>
          </div>

          <label className="field" htmlFor="distance">
            <span className="field__label">距离（km，允许 0.1 – 100）</span>
            <span className="input-wrap">
              <input
                id="distance"
                className="native-input input--suffix"
                type="number"
                step="0.1"
                min={0.1}
                max={100}
                value={distanceInput}
                onChange={(e) => setDistanceInput(e.target.value)}
                placeholder="例如 3.2"
              />
              <span className="input-suffix">km</span>
            </span>
          </label>

          <div className="quick-picks">
            {QUICK_DISTANCES.map((q) => (
              <button
                type="button"
                key={q.value}
                className="quick-pick"
                onClick={() => setDistanceInput(q.value)}
              >
                {q.label}
              </button>
            ))}
          </div>

          {distanceError !== '' && <p className="field-error">{distanceError}</p>}

          <label className="field" htmlFor="startedAt">
            <span className="field__label">出发时间（可选，默认当前）</span>
            <input
              id="startedAt"
              className="native-input"
              type="datetime-local"
              value={startedAt}
              onChange={(e) => setStartedAt(e.target.value)}
            />
          </label>

          {submitError !== '' && <p className="field-error">{submitError}</p>}

          <button type="submit" className="btn btn--primary btn--block">
            上报行程
          </button>
        </form>

        {last && (
          <div className="settle" style={{ marginTop: '1.5rem' }}>
            <div className="settle__head">
              <h2 className="settle__title">本次结果（后端核算）</h2>
              <span className="badge badge--ok">{last.cappedByDaily ? '已触单日上限' : '已入账'}</span>
            </div>
            <div className="settle__grid">
              <div>
                <p className="settle__label">出行方式</p>
                <p className="settle__value">
                  {modeEmoji(last.mode)} {modeLabel(last.mode)}
                </p>
                <p className="settle__note">{formatDistance(last.distanceKm)}</p>
              </div>
              <div>
                <p className="settle__label">碳减排量</p>
                <p className="settle__value">{formatGrams(last.carbonSavedGrams)}</p>
                <p className="settle__note">相对燃油私家车</p>
              </div>
              <div>
                <p className="settle__label">本次发放</p>
                <p className="settle__value settle__value--amber">
                  {formatPoints(last.grantedPoints)} 分
                </p>
                <p className="settle__note">原始 {formatPoints(last.rawPoints)} 分</p>
              </div>
            </div>
            <p className="settle__note">
              同一次出行重复提交不会重复发分，后端会返回首次结果。
            </p>
          </div>
        )}
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">最近行程</h2>
          <span className="muted">{tripsError ? '—' : `共 ${trips.length} 条`}</span>
        </div>

        {tripsError ? (
          <QueryErrorBlock error={tripsErrorObj} onRetry={() => void refetchTrips()} />
        ) : trips.length === 0 ? (
          <div className="empty">
            <span className="empty__art">
              <IconEmptyRoute size={48} />
            </span>
            <p className="empty__title">还没有行程记录</p>
            <p className="empty__note">选好出行方式、填上距离，就能拿到第一笔碳积分。</p>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => document.getElementById('distance')?.focus()}
            >
              上报第一程
            </button>
          </div>
        ) : (
          <div className="list">
            {trips.map((t) => (
              <div className="list__row" key={t.id}>
                <span className="list__icon">
                  <IconRoute size={16} />
                </span>
                <div className="list__body">
                  <p className="list__title">
                    {modeEmoji(t.mode)} {modeLabel(t.mode)} · {formatDistance(t.distanceKm)}
                  </p>
                  <p className="list__meta">
                    {formatDateTime(t.startedAt)} · 减排 {formatGrams(t.carbonSavedGrams)}
                  </p>
                </div>
                <span className="badge badge--ok">{t.status === 'VALID' ? '有效' : '已撤销'}</span>
                <span className="list__value list__value--amber">+{formatPoints(t.grantedPoints)}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
