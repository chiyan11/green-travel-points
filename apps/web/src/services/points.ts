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
        t.type === 'EARN' &&
        new Date(t.createdAt).getMonth() === now.getMonth() &&
        new Date(t.createdAt).getFullYear() === now.getFullYear(),
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
