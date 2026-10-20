import { useState, type FormEvent, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { TRAVEL_MODES, type TravelMode, type TripRecord } from '@/types/domain'
import { validateDistance } from '@/lib/validators'
import { listTrips, submitTrip } from '@/services/trips'
import { formatDateTime, formatDistance, formatGrams } from '@/lib/format'

function modeMeta(mode: TravelMode) {
  return TRAVEL_MODES.find((m) => m.code === mode) ?? { label: mode, emoji: '❓' }
}

export function TripsPage(): ReactNode {
  const queryClient = useQueryClient()
  const { data: trips = [] } = useQuery({
    queryKey: ['trips'],
    queryFn: () => Promise.resolve(listTrips()),
  })

  const [mode, setMode] = useState<TravelMode>('subway')
  const [distanceInput, setDistanceInput] = useState('')
  const [traveledAt, setTraveledAt] = useState('')
  const [distanceError, setDistanceError] = useState('')
  const [last, setLast] = useState<TripRecord | null>(null)
  const [submitError, setSubmitError] = useState('')

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setSubmitError('')
    const check = validateDistance(distanceInput)
    if (!check.ok) {
      setDistanceError(check.message)
      return
    }
    setDistanceError('')
    try {
      const trip = submitTrip({
        mode,
        distanceKm: check.value,
        traveledAt: traveledAt ? new Date(traveledAt).toISOString() : new Date().toISOString(),
      })
      setLast(trip)
      setDistanceInput('')
      setTraveledAt('')
      void queryClient.invalidateQueries({ queryKey: ['trips'] })
      void queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['txns'] })
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : '提交失败')
    }
  }

  return (
    <>
      <section className="hero">
        <h2>行程上报</h2>
        <p>选择出行方式、填写距离，系统核算碳减排量并折算积分（前端为预览，最终以后端计算为准）。</p>
      </section>

      <section className="card">
        <h3>提交一次出行</h3>
        <form onSubmit={onSubmit}>
          <div className="form-field">
            <label>出行方式</label>
            <div className="mode-grid">
              {TRAVEL_MODES.map((m) => (
                <label key={m.code} className="mode-option">
                  <input
                    type="radio"
                    name="mode"
                    value={m.code}
                    checked={mode === m.code}
                    onChange={() => setMode(m.code)}
                  />
                  <span>{m.emoji} {m.label}</span>
                </label>
              ))}
            </div>
          </div>

          <div className="form-field">
            <label htmlFor="distance">距离（km，允许 0.1 – 100）</label>
            <input
              id="distance"
              type="number"
              step="0.1"
              min={0.1}
              max={100}
              value={distanceInput}
              onChange={(e) => setDistanceInput(e.target.value)}
              placeholder="例如 3.2"
            />
            {distanceError && <p className="field-error">{distanceError}</p>}
          </div>

          <div className="form-field">
            <label htmlFor="traveledAt">出行时间（可选，默认当前）</label>
            <input
              id="traveledAt"
              type="datetime-local"
              value={traveledAt}
              onChange={(e) => setTraveledAt(e.target.value)}
            />
          </div>

          {submitError && <p className="field-error">{submitError}</p>}
          <button type="submit" className="btn">上报行程</button>
        </form>

        {last && (
          <div className="result-panel">
            <h3 style={{ marginTop: 0 }}>本次结果（本地预览）</h3>
            <p>
              {modeMeta(last.mode).emoji} {modeMeta(last.mode).label} · {formatDistance(last.distanceKm)}
              ，减排 {formatGrams(last.carbonSavedGrams)}，原始 {last.rawPoints} 分，实发{' '}
              <b>{last.grantedPoints}</b> 分
              {last.cappedByDaily && '（已触单日 500 分上限）'}。
            </p>
          </div>
        )}
      </section>

      <section className="card">
        <h3>最近行程</h3>
        {trips.length === 0 ? (
          <p className="empty-state">还没有行程，先上报一次吧。</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>方式</th>
                <th>距离</th>
                <th>减排量</th>
                <th>原始/实发</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {trips.map((t) => (
                <tr key={t.id}>
                  <td>{formatDateTime(t.traveledAt)}</td>
                  <td>
                    {modeMeta(t.mode).emoji} {modeMeta(t.mode).label}
                  </td>
                  <td>{formatDistance(t.distanceKm)}</td>
                  <td>{formatGrams(t.carbonSavedGrams)}</td>
                  <td>
                    {t.rawPoints} / <b>{t.grantedPoints}</b>
                  </td>
                  <td>
                    <span className="badge badge--ok">已发放</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  )
}
