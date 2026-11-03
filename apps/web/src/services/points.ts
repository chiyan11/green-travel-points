/**
 * 积分服务（MOCK）。
 * TODO 联调替换：GET /api/v1/points/summary、GET /api/v1/points/transactions。
 */
import { mockDb } from '@/mock/data'
import { DAILY_POINTS_CAP, type PointSummary, type PointTxn } from '@/types/domain'

export function getSummary(): PointSummary {
  const trips = mockDb.getTrips()
  const totalCarbon = trips.reduce((sum, t) => sum + t.carbonSavedGrams, 0)
  const now = new Date()
  const earnedThisMonth = mockDb
    .getTxns()
    .filter(
      (t) =>
        t.type === 'TRAVEL' &&
        new Date(t.occurredAt).getMonth() === now.getMonth() &&
        new Date(t.occurredAt).getFullYear() === now.getFullYear(),
    )
    .reduce((sum, t) => sum + t.amount, 0)

  return {
    balance: mockDb.getBalance(),
    totalCarbonSavedGrams: totalCarbon,
    earnedThisMonth,
    earnedToday: mockDb.earnedToday(),
    dailyCap: DAILY_POINTS_CAP,
  }
}

export function listTxns(): PointTxn[] {
  return mockDb.getTxns()
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
