/**
 * 前端领域类型与常量（手写契约版）。
 *
 * 后端 OpenAPI（docs/api/openapi.yaml，P1+P2 Sprint 2 交付）就绪后，
 * 改用 openapi-typescript 生成，本文件仅作过渡期占位，字段命名以后端为准。
 * 常量口径与 apps/api/internal/domain、db/migrations/000001 种子保持一致。
 */

export type TravelMode = 'walk' | 'bike' | 'subway' | 'bus' | 'shared_bike'

export interface TravelModeMeta {
  code: TravelMode
  label: string
  /** 实际排放因子 gCO2e/km，对应 travel_mode.ef_actual 种子值 */
  efActual: number
  emoji: string
}

/** 与 travel_mode 表种子一致：步/骑/共享单车=0，地铁 41，公交 68。 */
export const TRAVEL_MODES: readonly TravelModeMeta[] = [
  { code: 'walk', label: '步行', efActual: 0, emoji: '🚶' },
  { code: 'bike', label: '骑行', efActual: 0, emoji: '🚴' },
  { code: 'subway', label: '地铁', efActual: 41, emoji: '🚇' },
  { code: 'bus', label: '公交', efActual: 68, emoji: '🚌' },
  { code: 'shared_bike', label: '共享单车', efActual: 0, emoji: '🚲' },
]

/** 燃油私家车基准排放因子 gCO2e/km（EmissionFactorBase = 192）。 */
export const EMISSION_FACTOR_BASE = 192
/** 多少克碳减排兑换 1 积分（sys_config.carbon_per_point = 100）。 */
export const GRAMS_PER_POINT = 100
/** 单日积分上限（sys_config.daily_points_cap = 500，0 表示不限）。 */
export const DAILY_POINTS_CAP = 500
/** 单次行程距离合法区间 km（含端点）。 */
export const MIN_DISTANCE_KM = 0.1
export const MAX_DISTANCE_KM = 100

// ===== 接口 DTO（字段为草案，待 OpenAPI 校准） =====

export type TripStatus = 'VERIFIED' | 'REJECTED'

/** POST /api/v1/trips 的响应 / 行程列表项。 */
export interface TripRecord {
  id: string
  mode: TravelMode
  distanceKm: number
  traveledAt: string
  carbonSavedGrams: number
  rawPoints: number
  grantedPoints: number
  cappedByDaily: boolean
  status: TripStatus
}

export interface PointSummary {
  balance: number
  totalCarbonSavedGrams: number
  earnedThisMonth: number
  earnedToday: number
  dailyCap: number
}

export type PointTxnType = 'EARN' | 'SPEND' | 'EXPIRE' | 'ADJUST' | 'REVERSE'

/** point_txn 流水，append-only；amount 正为获得、负为消耗。 */
export interface PointTxn {
  id: string
  type: PointTxnType
  amount: number
  balanceAfter: number
  reason: string
  refType: 'TRIP' | 'ORDER' | 'ADMIN' | 'EXPIRE'
  refId: string
  createdAt: string
}

export interface Product {
  id: string
  name: string
  description: string
  pointsPrice: number
  stock: number
  category: string
  imageEmoji: string
  active: boolean
  /** 单用户限兑数量，缺省不限。 */
  limitPerUser?: number
}

export type OrderStatus = 'PENDING' | 'PAID' | 'SHIPPED' | 'COMPLETED' | 'CANCELLED'

export interface RedeemOrder {
  id: string
  productId: string
  productName: string
  pointsSpent: number
  status: OrderStatus
  receiver: string
  createdAt: string
  updatedAt: string
}

export interface AuthUser {
  userId: string
  nickname: string
  token: string
}

/** 分页响应壳（草案）。 */
export interface Page<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}
