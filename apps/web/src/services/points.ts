/**
 * 积分服务。
 * GET /api/v1/points/summary、GET /api/v1/points/transactions（需登录）。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成）；`bizRefType` /
 * `occurredAt` 是页面视图模型（types/domain.ts）的字段名，wire 上是 `refType` / `createdAt`。
 */
import { api } from '@/lib/http'
import type { PointTxn as WirePointTxn, PointTxnPage, PointsSummary } from '@/types/api'
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
 * 近 7 日碳减排趋势（原型：固定演示序列）。
 * TODO 联调替换：GET /api/v1/points/daily-carbon?days=7（契约已在 openapi.yaml 中，
 * 需要把调用方（数据看板）改成异步查询后再接入，见 docs 待办）。
 */
export function getRecentCarbonSeries(): DailyCarbonPoint[] {
  const demoGrams = [320, 483, 0, 960, 640, 483, 992]
  const out: DailyCarbonPoint[] = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date()
    d.setDate(d.getDate() - i)
    out.push({
      date: `${d.getMonth() + 1}/${d.getDate()}`,
      grams: demoGrams[6 - i] ?? 0,
    })
  }
  return out
}
