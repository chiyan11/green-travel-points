/**
 * 行程服务。
 * GET /api/v1/trips、POST /api/v1/trips（需登录）。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成）。映射时处理两处口径差异：
 * - 后端 wire 字段是 `traveledAt`，页面视图模型用 `startedAt`（对应 trip_record.started_at）
 * - `rawPoints > grantedPoints` 即被单日上限截断，`cappedByDaily` 缺失时由两者推算
 * 积分与碳减排一律以后端 domain.EvaluateTrip 为准，前端不再本地计算。
 */
import { api } from '@/lib/http'
import type { Trip as WireTrip, TripPage, TripStatus as WireTripStatus } from '@/types/api'
import type { TravelMode, TripRecord, TripStatus } from '@/types/domain'

export interface SubmitTripInput {
  mode: TravelMode
  distanceKm: number
  /** ISO 8601 字符串，随请求体以 traveledAt 发送。 */
  startedAt: string
}

function normalizeStatus(status: WireTripStatus): TripStatus {
  // POST /api/v1/trips 目前硬编码返回 "VERIFIED"（见 openapi.yaml 的 TripStatus 说明），
  // 列表返回库里的 VALID / REVOKED；这里收敛到 trip_record.status 的两个枚举值。
  return status === 'REVOKED' ? 'REVOKED' : 'VALID'
}

function toTrip(r: WireTrip): TripRecord {
  return {
    id: r.id,
    mode: r.mode,
    distanceKm: r.distanceKm,
    startedAt: r.traveledAt,
    carbonSavedGrams: r.carbonSavedGrams,
    rawPoints: r.rawPoints,
    grantedPoints: r.grantedPoints,
    cappedByDaily: r.cappedByDaily,
    status: normalizeStatus(r.status),
  }
}

export async function listTrips(): Promise<TripRecord[]> {
  const page = await api.get<TripPage>('/api/v1/trips')
  return (page.items ?? []).map(toTrip)
}

/** 上报行程：后端做幂等（同用户+同模式+同距离+同时间只记一次，重复提交返回首次结果）。 */
export async function submitTrip(input: SubmitTripInput): Promise<TripRecord> {
  const raw = await api.post<WireTrip>('/api/v1/trips', {
    mode: input.mode,
    distanceKm: input.distanceKm,
    traveledAt: input.startedAt || new Date().toISOString(),
  })
  return toTrip(raw)
}
