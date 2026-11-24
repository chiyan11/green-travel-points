import { useState, type FormEvent, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Badge } from '@astryxdesign/core/Badge'
import { Button } from '@astryxdesign/core/Button'
import { Card } from '@astryxdesign/core/Card'
import { Field } from '@astryxdesign/core/Field'
import { VStack } from '@astryxdesign/core/VStack'
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

      <Card padding={5} elevation="low">
        <div className="section-head">
          <h3>提交一次出行</h3>
          <span className="muted">单日最多计 500 分</span>
        </div>

        <form onSubmit={onSubmit}>
          <VStack gap={4}>
            <div className="form-field">
              <span className="field-label">出行方式</span>
              <div className="mode-grid">
                {TRAVEL_MODES.map((m) => (
                  <label key={m.code} className={mode === m.code ? 'mode-option mode-option--active' : 'mode-option'}>
                    <input
                      type="radio"
                      name="mode"
                      value={m.code}
                      checked={mode === m.code}
                      onChange={() => setMode(m.code)}
                    />
                    <span>
                      {m.emoji} {m.label}
                    </span>
                  </label>
                ))}
              </div>
            </div>

            <Field label="距离（km，允许 0.1 – 100）" inputID="distance">
              <input
                id="distance"
                className="native-input"
                type="number"
                step="0.1"
                min={0.1}
                max={100}
                value={distanceInput}
                onChange={(e) => setDistanceInput(e.target.value)}
                placeholder="例如 3.2"
              />
            </Field>
            {distanceError !== '' && <p className="field-error">{distanceError}</p>}

            <Field label="出发时间（可选，默认当前）" inputID="startedAt">
              <input
                id="startedAt"
                className="native-input"
                type="datetime-local"
                value={startedAt}
                onChange={(e) => setStartedAt(e.target.value)}
              />
            </Field>

            {submitError !== '' && <p className="field-error">{submitError}</p>}

            <div>
              <Button label="上报行程" type="submit" variant="primary" />
            </div>
          </VStack>
        </form>

        {last && (
          <div className="result-panel">
            <h3 style={{ marginTop: 0 }}>本次结果（后端核算）</h3>
            <div className="result-grid">
              <div>
                <p className="stat-card__label">出行方式</p>
                <p className="stat-card__value">{`${modeMeta(last.mode).emoji} ${modeMeta(last.mode).label}`}</p>
                <p className="stat-card__hint">{formatDistance(last.distanceKm)}</p>
              </div>
              <div>
                <p className="stat-card__label">碳减排量</p>
                <p className="stat-card__value">{formatGrams(last.carbonSavedGrams)}</p>
                <p className="stat-card__hint">相对燃油私家车</p>
              </div>
              <div>
                <p className="stat-card__label">发放积分</p>
                <p className="stat-card__value">{last.grantedPoints} 分</p>
                <p className="stat-card__hint">
                  原始 {last.rawPoints} 分{last.cappedByDaily ? ' · 已触单日上限' : ''}
                </p>
              </div>
            </div>
            <p className="muted" style={{ marginBottom: 0 }}>
              同一次出行重复提交不会重复发分（后端按用户+方式+距离+时间做幂等，返回首次结果）。
            </p>
          </div>
        )}
      </Card>

      <Card padding={5} elevation="low">
        <div className="section-head">
          <h3>最近行程</h3>
          <span className="muted">共 {trips.length} 条</span>
        </div>
        {trips.length === 0 ? (
          <p className="empty-state">还没有行程，先上报一次吧。</p>
        ) : (
          <div className="table-wrap">
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
                    <td>{formatDateTime(t.startedAt)}</td>
                    <td>
                      {modeMeta(t.mode).emoji} {modeMeta(t.mode).label}
                    </td>
                    <td>{formatDistance(t.distanceKm)}</td>
                    <td>{formatGrams(t.carbonSavedGrams)}</td>
                    <td>
                      {t.rawPoints} / <b>{t.grantedPoints}</b>
                    </td>
                    <td>
                      <Badge
                        label={t.status === 'VALID' ? '有效' : '已撤销'}
                        variant={t.status === 'VALID' ? 'success' : 'neutral'}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </>
  )
}
