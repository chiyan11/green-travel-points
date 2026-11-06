/**
 * 前端领域类型与常量（手写契约版）。
 *
 * 后端 OpenAPI（docs/api/openapi.yaml）就绪后，改用 openapi-typescript 生成，本文件仅作过渡期占位。
 * 字段口径已与 db/migrations/000003（积分账本）/000004（用户/行程/碳记录）/000005（商城/订单）逐字段对齐：
 * 枚举值、列名 snake_case→camelCase 映射以后端表结构为准。
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

// ===== 接口 DTO（已对齐 db/migrations；后端响应 JSON 用 camelCase） =====

/** trip_record.status（000004）：VALID 有效 / REVOKED 已撤销。 */
export type TripStatus = 'VALID' | 'REVOKED'

/**
 * 行程记录项。对应 trip_record + carbon_record：
 * startedAt=started_at，carbonSavedGrams=carbon_record.carbon_saved_g。
 * rawPoints / grantedPoints / cappedByDaily 为本地预览派生值，是否由列表接口返回待后端确认。
 */
export interface TripRecord {
  id: string
  mode: TravelMode
  distanceKm: number
  startedAt: string
  carbonSavedGrams: number
  rawPoints: number
  grantedPoints: number
  cappedByDaily: boolean
  status: TripStatus
}

/** 积分账户摘要。balance=frozen 是否由接口一并返回待确认（可兑余额 = balance - frozen）。 */
export interface PointSummary {
  balance: number
  totalCarbonSavedGrams: number
  earnedThisMonth: number
  earnedToday: number
  dailyCap: number
}

/** point_txn.txn_type（000003）：行程发放/兑换消耗/取消返还/人工调整/过期。 */
export type PointTxnType = 'TRAVEL' | 'REDEEM' | 'REFUND' | 'ADJUST' | 'EXPIRE'

/** point_txn.biz_ref_type（000003 注释）：trip_record / redeem_order / audit_log。 */
export type PointBizRefType = 'trip_record' | 'redeem_order' | 'audit_log'

/** 积分流水（append-only 账本）；amount 正为入账、负为出账，occurredAt=occurred_at。 */
export interface PointTxn {
  id: string
  type: PointTxnType
  amount: number
  balanceAfter: number
  bizRefType: PointBizRefType
  bizRefId: string
  occurredAt: string
}

/** product.status（000005）：DRAFT 草稿 / ONLINE 上架 / OFFLINE 下架。 */
export type ProductStatus = 'DRAFT' | 'ONLINE' | 'OFFLINE'

/**
 * 商品展示项。DB 结构为 product 1—N product_sku（pointsPrice/stock 在 SKU 上），
 * 列表接口先拍平取默认 SKU；多规格选品交互待后端定 SKU 查询参数。
 * coverUrl=product.cover_url；mock 阶段为空串，页面按分类兜底图标。
 */
export interface Product {
  id: string
  name: string
  description: string
  category: string
  coverUrl: string
  status: ProductStatus
  pointsPrice: number
  stock: number
  /** 单用户限兑数量，0 表示不限。 */
  limitPerUser: number
}

/** redeem_order.status（000005 / 10-状态图-订单）。 */
export type OrderStatus = 'PENDING' | 'PAID' | 'SHIPPED' | 'COMPLETED' | 'CANCELLED'

/** 兑换订单。orderNo=对外订单号，redeemCode=核销码（SHIPPED 后生效），quantity 默认 1。 */
export interface RedeemOrder {
  id: string
  orderNo: string
  productId: string
  skuId: string
  skuName: string
  productName: string
  pointsSpent: number
  quantity: number
  receiver: string
  status: OrderStatus
  redeemCode: string
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
