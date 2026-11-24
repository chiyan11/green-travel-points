/**
 * 积分服务。
 * GET /api/v1/points/summary、GET /api/v1/points/transactions（需登录）。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成）；`bizRefType` /
 * `occurredAt` 是页面视图模型（types/domain.ts）的字段名，wire 上是 `refType` / `createdAt`。
 */
import { api } from '@/lib/http'
import type {
  DailyCarbonPoint as WireDailyCarbonPoint,
  PointTxn as WirePointTxn,
  PointTxnPage,
  PointsSummary,
} from '@/types/api'
import {
  DAILY_POINTS_CAP,
  type PointBizRefType,
  type PointSummary,
  type PointTxn,
  type PointTxnType,
} from '@/types/domain'

function normalizeRefType(r: WirePointTxn): PointBizRefType {
  const refType: string = r.refType
  if (refType === 'trip_record' || refType === 'redeem_order' || refType === 'audit_log') return refType
  return 'audit_log'
}

function toTxn(r: WirePointTxn): PointTxn {
  return {
    id: String(r.id),
    type: r.type as PointTxnType,
    amount: r.amount,
    balanceAfter: r.balanceAfter,
    bizRefType: normalizeRefType(r),
    bizRefId: String(r.refId),
    occurredAt: r.createdAt,
  }
}

export async function getSummary(): Promise<PointSummary> {
  const s = await api.get<PointsSummary>('/api/v1/points/summary')
  return {
    balance: s.balance,
    totalCarbonSavedGrams: s.totalCarbonSavedGrams,
    earnedThisMonth: s.earnedThisMonth,
    earnedToday: s.earnedToday,
    dailyCap: s.dailyCap || DAILY_POINTS_CAP,
  }
}

export async function listTxns(): Promise<PointTxn[]> {
  const page = await api.get<PointTxnPage>('/api/v1/points/transactions?page=1&pageSize=50')
  return (page.items ?? []).map(toTxn)
}

export interface DailyCarbonPoint {
  /** M/D，展示用 */
  date: string
  /** 当日碳减排 gCO2e */
  grams: number
}

/**
 * 近 N 日碳减排趋势：GET /api/v1/points/daily-carbon（缺失日期后端已补 0）。
 * wire 上的 date 是 `YYYY-MM-DD`，页面只需要 `M/D`。
 */
export async function getCarbonSeries(days = 7): Promise<DailyCarbonPoint[]> {
  const rows = await api.get<WireDailyCarbonPoint[]>(`/api/v1/points/daily-carbon?days=${days}`)
  return rows.map((r) => ({ date: toMonthDay(r.date), grams: r.grams }))
}

function toMonthDay(date: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(date)
  if (!m) return date
  return `${Number(m[2])}/${Number(m[3])}`
}
