/**
 * 行程服务（MOCK，本地预览计算）。
 * TODO 联调替换：GET/POST /api/v1/trips，碳减排与积分以后端 EvaluateTrip 为准。
 */
import { mockDb, nextId } from '@/mock/data'
import { capDaily, carbonSavedGrams, rawPointsFor } from '@/lib/validators'
import { TRAVEL_MODES, type TravelMode, type TripRecord } from '@/types/domain'

export interface SubmitTripInput {
  mode: TravelMode
  distanceKm: number
  traveledAt: string
}

export function listTrips(): TripRecord[] {
  return mockDb.getTrips()
}

function modeLabel(mode: TravelMode): string {
  return TRAVEL_MODES.find((m) => m.code === mode)?.label ?? mode
}

export function submitTrip(input: SubmitTripInput): TripRecord {
  const saved = carbonSavedGrams(input.distanceKm, input.mode)
  const raw = rawPointsFor(saved)
  const granted = capDaily(raw, mockDb.earnedToday())

  const trip: TripRecord = {
    id: nextId('trip'),
    mode: input.mode,
    distanceKm: input.distanceKm,
    traveledAt: input.traveledAt || new Date().toISOString(),
    carbonSavedGrams: saved,
    rawPoints: raw,
    grantedPoints: granted,
    cappedByDaily: granted < raw,
    status: 'VERIFIED',
  }
  mockDb.addTrip(trip)

  if (granted > 0) {
    mockDb.addPoints(granted)
    mockDb.addTxn({
      id: nextId('txn'),
      type: 'EARN',
      amount: granted,
      balanceAfter: mockDb.getBalance(),
      reason: `${modeLabel(input.mode)} ${input.distanceKm}km 出行奖励`,
      refType: 'TRIP',
      refId: trip.id,
      createdAt: new Date().toISOString(),
    })
  }
  return trip
}
