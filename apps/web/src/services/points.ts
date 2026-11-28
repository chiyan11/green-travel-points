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

const KNOWN_TXN_TYPE: readonly string[] = ['TRAVEL', 'REDEEM', 'REFUND', 'ADJUST', 'EXPIRE']
const KNOWN_REF_TYPE: readonly string[] = ['trip_record', 'redeem_order', 'audit_log']

/** 未知枚举值落到 UNKNOWN（原始值另行保留），不再硬塞成 audit_log 改掉语义。 */
function normalizeTxnType(raw: string | undefined): PointTxnType {
  const value = (raw ?? '').toUpperCase()
  return KNOWN_TXN_TYPE.includes(value) ? (value as PointTxnType) : 'UNKNOWN'
}

function normalizeRefType(raw: string | undefined): PointBizRefType {
  const value = raw ?? ''
  return KNOWN_REF_TYPE.includes(value) ? (value as PointBizRefType) : 'UNKNOWN'
}

function toTxn(r: WirePointTxn): PointTxn {
  return {
    id: String(r.id),
    type: normalizeTxnType(r.type),
    typeText: r.type ?? '',
    amount: r.amount,
    balanceAfter: r.balanceAfter,
    bizRefType: normalizeRefType(r.refType),
    bizRefTypeText: r.refType ?? '',
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
    // `||` 会把后端的 0（不限上限）误当成「没给值」翻成 500
    dailyCap: s.dailyCap ?? DAILY_POINTS_CAP,
  }
}

export interface TxnList {
  items: PointTxn[]
  page: number
  pageSize: number
  total: number
}

export async function listTxns(page = 1, pageSize = 50): Promise<TxnList> {
  const res = await api.get<PointTxnPage>(
    `/api/v1/points/transactions?page=${page}&pageSize=${pageSize}`,
  )
  return {
    items: (res.items ?? []).map(toTxn),
    page: res.page,
    pageSize: res.pageSize,
    total: res.total,
  }
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
