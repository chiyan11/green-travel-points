import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { CarbonTrendChart } from '@/components/CarbonTrendChart'
import { QueryErrorBlock } from '@/components/QueryError'
import { IconAward, IconTrendDown, IconTrendUp } from '@/components/icons'
import { pointTxnTypeLabel } from '@/lib/orderStatus'
import { formatDateTime, formatGrams, formatPoints } from '@/lib/format'
import { getCarbonSeries, getSummary, listTxns } from '@/services/points'
import type { PointBizRefType } from '@/types/domain'

/** 业务引用类型的中文说明；未知取值回落到后端原始串，不吞信息。 */
const BIZ_LABEL: Record<PointBizRefType, string> = {
  trip_record: '行程核算',
  redeem_order: '商城兑换',
  audit_log: '人工调整',
  UNKNOWN: '其他来源',
}

function bizLabel(type: PointBizRefType, raw: string): string {
  return type === 'UNKNOWN' && raw.trim() !== '' ? raw : BIZ_LABEL[type]
}

/** 我的积分：一张主卡放余额与今日额度，两张副卡放累计减排与本月获得。 */
export function PointsPage(): ReactNode {
  const {
    data: summary,
    isError: summaryError,
    error: summaryErrorObj,
    refetch: refetchSummary,
  } = useQuery({ queryKey: ['summary'], queryFn: () => getSummary() })
  const {
    data: txnList,
    isPending: txnsPending,
    isError: txnsError,
    error: txnsErrorObj,
    refetch: refetchTxns,
  } = useQuery({
    queryKey: ['txns'],
    queryFn: () => listTxns(),
  })
  const txns = txnList?.items ?? []
  const txnsTotal = txnList?.total ?? 0

  const {
    data: trend = [],
    isPending: trendPending,
    isError: trendError,
    error: trendErrorObj,
    refetch: refetchTrend,
  } = useQuery({
    queryKey: ['carbon-trend-7d'],
    queryFn: () => getCarbonSeries(7),
  })

  const balance = summary?.balance ?? 0
  const earnedToday = summary?.earnedToday ?? 0
  const cap = summary?.dailyCap ?? 500
  // 后端的 0 表示不限上限，不能显示成「单日上限 0 分」
  const capText = cap === 0 ? '不限' : `${formatPoints(cap)} 分`
  const pct = cap === 0 ? 0 : Math.min(100, Math.round((earnedToday / Math.max(cap, 1)) * 100))

  return (
    <div className="page">
      <div className="page__head">
        <div>
          <h1 className="page-title">我的积分</h1>
          <p className="page-sub">余额与流水只通过账本变更，当日发放受上限约束。</p>
        </div>
        <span className="badge badge--muted">单日上限 {capText}</span>
      </div>

      {summaryError ? (
        <section className="card">
          <QueryErrorBlock error={summaryErrorObj} onRetry={() => void refetchSummary()} />
        </section>
      ) : (
      <div className="stat-row">
        <section className="stat stat--main stat-card">
          <div className="stat__main-row">
            <div>
              <p className="stat__label">当前余额</p>
              <p className="stat__value">
                {formatPoints(balance)}
                <span className="stat__unit"> 分</span>
              </p>
              <p className="stat__note">可兑换商城商品</p>
            </div>
            <div className="stat__ring">
              <span className="ring" style={{ ['--pct' as string]: pct }}>
                <span className="ring__inner">
                  <span className="ring__value">{pct}%</span>
                </span>
              </span>
              <span className="ring__caption">今日额度</span>
            </div>
          </div>
          <p className="stat__note">
            {cap === 0
              ? `今日已发 ${formatPoints(earnedToday)} 分，当前不设上限`
              : `今日已发 ${formatPoints(earnedToday)} / ${formatPoints(cap)} 分，超出部分不计积分`}
          </p>
        </section>

        <section className="stat stat-card">
          <p className="stat__label">累计碳减排</p>
          <p className="stat__value">{formatGrams(summary?.totalCarbonSavedGrams ?? 0)}</p>
          <p className="stat__note">全部行程累计，按 192 g/km 基准核算</p>
        </section>

        <section className="stat stat-card">
          <p className="stat__label">本月获得</p>
          <p className="stat__value">{formatPoints(summary?.earnedThisMonth ?? 0)} 分</p>
          <p className="stat__note">按自然月统计</p>
        </section>
      </div>
      )}

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">近 7 日碳减排</h2>
          <span className="muted">单位 gCO₂e</span>
        </div>
        {trendPending ? (
          <span className="skeleton skeleton--block" style={{ width: '100%', height: 180 }} />
        ) : trendError ? (
          <QueryErrorBlock error={trendErrorObj} onRetry={() => void refetchTrend()} />
        ) : (
          <CarbonTrendChart data={trend} />
        )}
      </section>

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">积分流水</h2>
          <span className="muted">
            {txnsPending
              ? '加载中…'
              : txnsError
                ? '—'
                : txnsTotal > txns.length
                  ? `共 ${txnsTotal} 条，当前显示最近 ${txns.length} 条`
                  : `共 ${txns.length} 条`}
          </span>
        </div>

        {txnsPending ? (
          <div className="list">
            {[0, 1, 2].map((i) => (
              <div className="list__row" key={i}>
                <span className="skeleton" style={{ width: 36, height: 36, borderRadius: '50%' }} />
                <div className="list__body">
                  <span className="skeleton skeleton--line" style={{ width: '28%' }} />
                  <span className="skeleton skeleton--line" style={{ width: '52%' }} />
                </div>
              </div>
            ))}
          </div>
        ) : txnsError ? (
          <QueryErrorBlock error={txnsErrorObj} onRetry={() => void refetchTxns()} />
        ) : txns.length === 0 ? (
          <div className="empty">
            <span className="empty__art">
              <IconAward size={40} />
            </span>
            <p className="empty__title">还没有流水</p>
            <p className="empty__note">上报一次出行，第一笔积分就会出现在这里。</p>
          </div>
        ) : (
          <div className="list">
            {txns.map((t) => {
              const meta = { label: pointTxnTypeLabel(t.type, t.typeText) }
              const isOut = t.amount < 0
              return (
                <div className="list__row" key={t.id}>
                  <span className={isOut ? 'list__icon list__icon--out' : 'list__icon'}>
                    {isOut ? <IconTrendDown size={16} /> : <IconTrendUp size={16} />}
                  </span>
                  <div className="list__body">
                    <p className="list__title">{meta.label}</p>
                    <p className="list__meta">
                      {bizLabel(t.bizRefType, t.bizRefTypeText)} · {formatDateTime(t.occurredAt)} · 余额{' '}
                      {formatPoints(t.balanceAfter)}
                    </p>
                  </div>
                  <span className={isOut ? 'list__value list__value--out' : 'list__value list__value--amber'}>
                    {t.amount >= 0 ? '+' : ''}
                    {formatPoints(t.amount)}
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </section>
    </div>
  )
}
