/**
 * 行程服务（联调版）。
 * GET /api/v1/trips、POST /api/v1/trips（需登录）。
 *
 * 字段兼容说明（后端 trips.go 与迁移表 000004 存在命名差异）：
 * - 后端 JSON 用 `traveledAt`，前端契约用 `startedAt`（started_at），这里双向对齐
 * - 上报响应里 status 硬编码为 "VERIFIED"，而库里 trip_record.status 是 VALID/REVOKED，
 *   统一归一成前端枚举，避免页面出现两种状态值
 * - 积分与碳减排一律以后端 domain.EvaluateTrip 为准，前端不再本地计算
 */
import { api } from '@/lib/http'
import type { Page, TravelMode, TripRecord, TripStatus } from '@/types/domain'

export interface SubmitTripInput {
  mode: TravelMode
  distanceKm: number
  /** ISO 8601 字符串，随请求体以 traveledAt 发送。 */
  startedAt: string
}

interface RawTrip {
  id: string
  mode: TravelMode
  distanceKm: number
  /** 后端字段名；兼容尚未改名的实现。 */
  traveledAt?: string
  startedAt?: string
  carbonSavedGrams: number
  rawPoints?: number
  grantedPoints?: number
  cappedByDaily?: boolean
  status?: string
}

function normalizeStatus(status: string | undefined): TripStatus {
  if (status === 'REVOKED' || status === 'REJECTED') return 'REVOKED'
  return 'VALID'
}

function toTrip(r: RawTrip): TripRecord {
  const raw = r.rawPoints ?? 0
  const granted = r.grantedPoints ?? 0
  return {
    id: r.id,
    mode: r.mode,
    distanceKm: r.distanceKm,
    startedAt: r.startedAt || r.traveledAt || '',
    carbonSavedGrams: r.carbonSavedGrams,
    rawPoints: raw,
    grantedPoints: granted,
    cappedByDaily: r.cappedByDaily ?? granted < raw,
    status: normalizeStatus(r.status),
  }
}

export async function listTrips(): Promise<TripRecord[]> {
  const page = await api.get<Page<RawTrip>>('/api/v1/trips')
  return (page.items ?? []).map(toTrip)
}

/** 上报行程：后端做幂等（同用户+同模式+同距离+同时间只记一次，重复提交返回首次结果）。 */
export async function submitTrip(input: SubmitTripInput): Promise<TripRecord> {
  const raw = await api.post<RawTrip>('/api/v1/trips', {
    mode: input.mode,
    distanceKm: input.distanceKm,
    traveledAt: input.startedAt || new Date().toISOString(),
  })
  return toTrip(raw)
}
