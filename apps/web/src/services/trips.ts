/**
 * 行程服务（MOCK，本地预览计算）。
 * TODO 联调替换：GET/POST /api/v1/trips，碳减排与积分以后端 EvaluateTrip 为准。
 * 字段口径对齐 trip_record（000004）：startedAt=started_at，status=VALID。
 */
import { mockDb, nextId } from '@/mock/data'
import { capDaily, carbonSavedGrams, rawPointsFor } from '@/lib/validators'
import { type TravelMode, type TripRecord } from '@/types/domain'

export interface SubmitTripInput {
  mode: TravelMode
  distanceKm: number
  startedAt: string
}

export function listTrips(): TripRecord[] {
  return mockDb.getTrips()
}

export function submitTrip(input: SubmitTripInput): TripRecord {
  const saved = carbonSavedGrams(input.distanceKm, input.mode)
  const raw = rawPointsFor(saved)
  const granted = capDaily(raw, mockDb.earnedToday())

  const trip: TripRecord = {
    id: nextId('trip'),
    mode: input.mode,
    distanceKm: input.distanceKm,
    startedAt: input.startedAt || new Date().toISOString(),
    carbonSavedGrams: saved,
    rawPoints: raw,
    grantedPoints: granted,
    cappedByDaily: granted < raw,
    status: 'VALID',
  }
  mockDb.addTrip(trip)

  if (granted > 0) {
    mockDb.addPoints(granted)
    mockDb.addTxn({
      id: nextId('txn'),
      type: 'TRAVEL',
      amount: granted,
      balanceAfter: mockDb.getBalance(),
      bizRefType: 'trip_record',
      bizRefId: trip.id,
      occurredAt: new Date().toISOString(),
    })
  }
  return trip
}
