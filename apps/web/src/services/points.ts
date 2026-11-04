/**
 * 积分服务（联调版）。
 * GET /api/v1/points/summary、GET /api/v1/points/transactions（需 X-User-Id）。
 * 流水字段做双向兼容：P3 当前返回 refType/refId/createdAt，迁移口径为 bizRefType/bizRefId/occurredAt。
 */
import { api } from '@/lib/http'
import {
  DAILY_POINTS_CAP,
  type Page,
  type PointBizRefType,
  type PointSummary,
  type PointTxn,
  type PointTxnType,
} from '@/types/domain'

interface RawSummary {
  balance: number
  frozen?: number
  available?: number
  totalCarbonSavedGrams?: number
  earnedThisMonth?: number
  earnedToday?: number
  dailyCap?: number
}

interface RawTxn {
  id: number
  type: string
  amount: number
  balanceAfter: number
  /** 过渡期：旧契约别名。 */
  refType?: string
  refId?: number
  createdAt?: string
  /** 迁移口径字段。 */
  bizRefType?: string
  bizRefId?: number
  occurredAt?: string
}

function normalizeRefType(t: RawTxn): PointBizRefType {
  if (t.bizRefType === 'trip_record' || t.bizRefType === 'redeem_order' || t.bizRefType === 'audit_log') {
    return t.bizRefType
  }
  switch (t.refType) {
    case 'TRIP':
      return 'trip_record'
    case 'ORDER':
      return 'redeem_order'
    default:
      return 'audit_log'
  }
}

function toTxn(r: RawTxn): PointTxn {
  return {
    id: String(r.id),
    type: (r.type || 'ADJUST') as PointTxnType,
    amount: r.amount,
    balanceAfter: r.balanceAfter,
    bizRefType: normalizeRefType(r),
    bizRefId: String(r.bizRefId ?? r.refId ?? ''),
    occurredAt: r.occurredAt || r.createdAt || '',
  }
}

export async function getSummary(): Promise<PointSummary> {
  const s = await api.get<RawSummary>('/api/v1/points/summary')
  return {
    balance: s.balance,
    totalCarbonSavedGrams: s.totalCarbonSavedGrams ?? 0,
    earnedThisMonth: s.earnedThisMonth ?? 0,
    earnedToday: s.earnedToday ?? 0,
    dailyCap: s.dailyCap ?? DAILY_POINTS_CAP,
  }
}

export async function listTxns(): Promise<PointTxn[]> {
  const page = await api.get<Page<RawTxn>>('/api/v1/points/transactions?page=1&pageSize=50')
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
 * TODO 联调替换：GET /api/v1/trips?from=<7天前>，由后端按日聚合 carbon_record.carbon_saved_g。
 * （/trips 为 P2 接口，尚未上线。）
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
