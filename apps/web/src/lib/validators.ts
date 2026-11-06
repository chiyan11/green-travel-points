/**
 * 前端侧校验与本地预览计算。
 * 规则口径与后端 internal/domain 对齐：前端只做即时反馈，最终以后端计算为准。
 */
import {
  DAILY_POINTS_CAP,
  EMISSION_FACTOR_BASE,
  GRAMS_PER_POINT,
  MAX_DISTANCE_KM,
  MIN_DISTANCE_KM,
  TRAVEL_MODES,
  type TravelMode,
} from '@/types/domain'

export type DistanceCheck =
  | { ok: true; value: number }
  | { ok: false; message: string }

/** 校验单次行程距离：非空、有限数字、落在 [0.1, 100] km（含端点）。 */
export function validateDistance(input: string): DistanceCheck {
  const trimmed = input.trim()
  if (trimmed === '') {
    return { ok: false, message: '请输入行程距离' }
  }
  const value = Number(trimmed)
  if (!Number.isFinite(value)) {
    return { ok: false, message: '距离必须是数字' }
  }
  if (value < MIN_DISTANCE_KM) {
    return { ok: false, message: `距离不能小于 ${MIN_DISTANCE_KM} km` }
  }
  if (value > MAX_DISTANCE_KM) {
    return { ok: false, message: `距离不能超过 ${MAX_DISTANCE_KM} km` }
  }
  return { ok: true, value }
}

export function isTravelMode(value: string): value is TravelMode {
  return TRAVEL_MODES.some((m) => m.code === value)
}

/** 本地预览：carbon_saved = max(0, d × (EF_base − EF_actual))。 */
export function carbonSavedGrams(distanceKm: number, mode: TravelMode): number {
  const meta = TRAVEL_MODES.find((m) => m.code === mode)
  if (!meta) return 0
  const saved = distanceKm * (EMISSION_FACTOR_BASE - meta.efActual)
  return saved > 0 ? saved : 0
}

/** 本地预览：floor(g / 100)，1e-9 容差吸收浮点误差（与 PointsFor 一致）。 */
export function rawPointsFor(carbonGrams: number): number {
  if (carbonGrams <= 0) return 0
  return Math.floor(carbonGrams / GRAMS_PER_POINT + 1e-9)
}

/** 本地预览：单日封顶 min(raw, cap − alreadyToday)，cap<=0 表示不限。 */
export function capDaily(raw: number, alreadyToday: number, cap: number = DAILY_POINTS_CAP): number {
  if (raw <= 0) return 0
  if (cap <= 0) return raw
  const remain = cap - Math.max(0, alreadyToday)
  if (remain <= 0) return 0
  return Math.min(raw, remain)
}
