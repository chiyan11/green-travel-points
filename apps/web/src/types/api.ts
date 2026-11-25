/* eslint-disable */
// 由 docs/api/openapi.yaml 生成（npm run gen:api），勿手改；契约变更请改 yaml 后重新生成。
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface AdjustPointsInput {
  /**
   * 正数=增加，负数=扣减，0 非法
   * @format int64
   */
  amount: number;
  /**
   * 操作理由（BR-14 必填）
   * @maxLength 255
   */
  reason: string;
  /** @format int64 */
  userId: number;
}

export interface AdjustResponse {
  code: "OK";
  data: AdjustResult;
  message: string;
}

export interface AdjustResult {
  /** @format int64 */
  accountId: number;
  /**
   * 对应的 audit_log.id
   * @format int64
   */
  auditId: number;
  /**
   * 调账后余额
   * @format int64
   */
  balance: number;
  /** @format int64 */
  userId: number;
}

export interface AdminProductInput {
  category?: string;
  description?: string;
  imageUrl?: string;
  limitPerUser?: number;
  /** @format int64 */
  merchantId?: number;
  name: string;
  skus: AdminSKUInput[];
  sortOrder?: number;
  status?: ProductStatus;
}

export interface AdminSKUInput {
  name: string;
  /** @format int64 */
  pointsPrice: number;
  stock: number;
}

/** 出行方式与排放因子（`travel_mode`） */
export interface AdminTravelMode {
  /** 出行方式代码，与 travel_mode 表种子一致 */
  code: TravelMode;
  /**
   * 该方式实际排放因子 gCO2e/km
   * @format double
   * @example 68
   */
  efActual: number;
  /**
   * 基准排放因子 gCO2e/km
   * @format double
   * @example 192
   */
  efBase: number;
  /** 停用后该方式上报行程返回 400 */
  enabled: boolean;
  /** @example "公交" */
  name: string;
}

export interface AuditLog {
  action: AuditLogAction;
  afterValue: string;
  /**
   * 积分变动量（仅调账非 0）
   * @format int64
   */
  amount: number;
  beforeValue: string;
  /** @format date-time */
  createdAt: string;
  /** @format int64 */
  id: number;
  /**
   * 操作人 user.id
   * @format int64
   */
  operatorId: number;
  reason: string;
  /** @format int64 */
  targetId: number;
  targetKey: string;
  targetType: AuditTargetType;
}

export type AuditLogAction = "ADJUST" | "CONFIG_UPDATE" | "TRAVEL_MODE_UPDATE";

export interface AuditLogPage {
  items: AuditLog[];
  page: number;
  pageSize: number;
  /** @format int64 */
  total: number;
}

export interface AuditLogPageResponse {
  code: "OK";
  data: AuditLogPage;
  message: string;
}

export type AuditTargetType = "point_account" | "sys_config" | "travel_mode";

export interface AuthSession {
  nickname: string;
  /** Bearer token，2 小时有效 */
  token: string;
  /**
   * 对外用户 ID，形如 `u_1005`
   * @example "u_1005"
   */
  userId: string;
}

export interface AuthSessionResponse {
  code: "OK";
  data: AuthSession;
  message: string;
}

export interface CancelOrderInput {
  reason?: string;
}

/** 一条系统参数（`sys_config`）；值一律为字符串，按 `valueType` 解析。 */
export interface ConfigItem {
  description: string;
  /** @example "daily_points_cap" */
  key: string;
  /** @example "500" */
  value: string;
  valueType: "string" | "int" | "decimal" | "bool";
}

export interface ConfigListResponse {
  code: "OK";
  data: ConfigItem[];
  message: string;
}

export interface ConfigUpdateInput {
  /**
   * 待修改参数。语义约束：`carbon_per_point > 0`、`points_expire_days > 0`、
   * `trip_min_distance_km > 0`、`trip_max_distance_km > 0`、`daily_points_cap >= 0`（0 = 不限制）；
   * 写入后复核 `trip_min_distance_km < trip_max_distance_km`。未知键或非法值返回 400。
   * @minItems 1
   */
  items: ConfigUpdateItem[];
}

export interface ConfigUpdateItem {
  /** 必须是 `sys_config` 里已有的参数键 */
  key: string;
  /** 字符串形态的新值，按该参数的 `valueType` 校验 */
  value: string;
}

export interface CreateOrderInput {
  /** @format int64 */
  productId: number;
  /** 收货人，缺省为「本人」 */
  receiver?: string;
  /**
   * 省略或 0 时由后端挑选第一个有库存的 SKU
   * @format int64
   */
  skuId?: number;
}

export interface DailyCarbonListResponse {
  code: "OK";
  data: DailyCarbonPoint[];
  message: string;
}

export interface DailyCarbonPoint {
  /**
   * 日期，形如 `2026-11-24`
   * @example "2026-11-24"
   */
  date: string;
  /** 当日碳减排克数 */
  grams: number;
}

/** 业务错误码（HTTP 状态码另行表达协议层语义） */
export type ErrorCode =
  | "INVALID_ARGUMENT"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "INTERNAL"
  | "UNAVAILABLE"
  | "INSUFFICIENT_POINTS"
  | "OUT_OF_STOCK"
  | "LIMIT_EXCEEDED"
  | "ILLEGAL_TRANSITION";

export interface ErrorEnvelope {
  /** 业务错误码（HTTP 状态码另行表达协议层语义） */
  code: ErrorCode;
  /** 面向人的中文错误说明，可直接展示 */
  message: string;
}

export interface ExpireResponse {
  code: "OK";
  /** 一次积分过期清算的结果 */
  data: ExpireResult;
  message: string;
}

/** 一次积分过期清算的结果 */
export interface ExpireResult {
  /**
   * 余额发生变化的账户数
   * @format int64
   */
  accounts: number;
  /**
   * 被置为 EXPIRED 的批次数
   * @format int64
   */
  batches: number;
  /**
   * 实际过期扣减的积分总数
   * @format int64
   */
  points: number;
}

export interface HealthResponse {
  code: "OK";
  data: HealthResult;
  message: string;
}

export interface HealthResult {
  /** @example "ok" */
  status: string;
}

export interface LoginInput {
  /** 手机号的历史别名，同时传时以 `account` 优先 */
  account?: string;
  password: string;
  phone?: string;
}

export interface LoginResponse {
  code: "OK";
  data: LoginResult;
  message: string;
}

export interface LoginResult {
  token: string;
  user: {
    nickname: string;
    userId: string;
  };
}

export interface MeResponse {
  code: "OK";
  data: MeResult;
  message: string;
}

export interface MeResult {
  /** @format date-time */
  joinedAt: string;
  nickname: string;
  userId: string;
}

export interface Order {
  cancelReason?: string;
  /** @format date-time */
  cancelledAt?: string;
  /** @format date-time */
  completedAt?: string;
  /** @format date-time */
  createdAt: string;
  /** @format int64 */
  id: number;
  orderNo: string;
  /** @format int64 */
  pointsSpent: number;
  /** @format int64 */
  productId: number;
  productName: string;
  quantity: number;
  receiver: string;
  /** 核销码，SHIPPED 之后可用，之前为空串 */
  redeemCode: string;
  /** @format date-time */
  shippedAt?: string;
  /** @format int64 */
  skuId: number;
  skuName: string;
  status: OrderStatus;
  /** @format date-time */
  updatedAt: string;
  /** @format int64 */
  userId: number;
}

export interface OrderPage {
  items: Order[];
  page: number;
  pageSize: number;
  /** @format int64 */
  total: number;
}

export interface OrderPageResponse {
  code: "OK";
  data: OrderPage;
  message: string;
}

export interface OrderResponse {
  code: "OK";
  data: Order;
  message: string;
}

export type OrderStatus =
  | "PENDING"
  | "PAID"
  | "SHIPPED"
  | "COMPLETED"
  | "CANCELLED";

export interface PingResponse {
  code: "OK";
  data: PingResult;
  message: string;
}

export interface PingResult {
  /** @example "dev" */
  env: string;
  /** @example "green-travel-points" */
  pong: string;
}

export type PointBizRefType = "trip_record" | "redeem_order" | "audit_log";

export interface PointTxn {
  /**
   * 正为入账，负为出账
   * @format int64
   */
  amount: number;
  /** @format int64 */
  balanceAfter: number;
  /** @format date-time */
  createdAt: string;
  /** @format int64 */
  id: number;
  reason: string;
  /** @format int64 */
  refId: number;
  refType: PointBizRefType;
  /** 行程发放 / 兑换消耗 / 取消返还 / 人工调整 / 过期 */
  type: PointTxnType;
}

export interface PointTxnPage {
  items: PointTxn[];
  page: number;
  pageSize: number;
  /** @format int64 */
  total: number;
}

export interface PointTxnPageResponse {
  code: "OK";
  data: PointTxnPage;
  message: string;
}

/** 行程发放 / 兑换消耗 / 取消返还 / 人工调整 / 过期 */
export type PointTxnType = "TRAVEL" | "REDEEM" | "REFUND" | "ADJUST" | "EXPIRE";

export interface PointsSummary {
  /**
   * 可兑余额 = balance - frozen
   * @format int64
   */
  available: number;
  /** @format int64 */
  balance: number;
  /**
   * 单日积分上限，0 表示不限
   * @format int64
   */
  dailyCap: number;
  /** @format int64 */
  earnedThisMonth: number;
  /** @format int64 */
  earnedToday: number;
  /** @format int64 */
  frozen: number;
  /** 累计碳减排克数（gCO2e） */
  totalCarbonSavedGrams: number;
}

export interface PointsSummaryResponse {
  code: "OK";
  data: PointsSummary;
  message: string;
}

export interface Product {
  /** 历史字段，等价于 `status == ONLINE`；新代码请用 status */
  active: boolean;
  category: string;
  description: string;
  /** @format int64 */
  id: number;
  /** 商品主图地址，缺省为空串（前端按分类兜底图标） */
  imageUrl: string;
  /** 单用户限兑数量，0 表示不限 */
  limitPerUser: number;
  /**
   * 商户 ID，前台接口可能省略
   * @format int64
   */
  merchantId?: number;
  name: string;
  /**
   * SKU 聚合最低价
   * @format int64
   */
  pointsPrice: number;
  /** 仅商品详情返回 */
  skus?: SKU[];
  status?: ProductStatus;
  /** SKU 聚合总库存 */
  stock: number;
}

export interface ProductPage {
  items: Product[];
  page: number;
  pageSize: number;
  /** @format int64 */
  total: number;
}

export interface ProductPageResponse {
  code: "OK";
  data: ProductPage;
  message: string;
}

export interface ProductResponse {
  code: "OK";
  data: Product;
  message: string;
}

export type ProductStatus = "DRAFT" | "ONLINE" | "OFFLINE";

/** 全部字段可选，未传的字段保持原值 */
export interface ProductUpdate {
  category?: string;
  description?: string;
  imageUrl?: string;
  limitPerUser?: number;
  name?: string;
  sortOrder?: number;
  status?: ProductStatus;
}

export interface ReadinessResponse {
  code: "OK";
  data: ReadinessResult;
  message: string;
}

export interface ReadinessResult {
  /** @example "up" */
  database: string;
  /** @example "ok" */
  status: string;
}

export interface ReconcileItem {
  /** @format int64 */
  accountId: number;
  /** @format int64 */
  balance: number;
  /** @format int64 */
  ledgerSum: number;
  /** @format int64 */
  userId: number;
}

export interface ReconcileResponse {
  code: "OK";
  data: ReconcileResult;
  message: string;
}

export interface ReconcileResult {
  /**
   * 参与对账的账户数
   * @format int64
   */
  accounts: number;
  mismatches: ReconcileItem[];
  ok: boolean;
}

/** `phone` 与 `account` 二者其一必填（`account` 是手机号的历史别名，同时传时以 `phone` 为准） */
export interface RegisterInput {
  account?: string;
  /** 缺省用手机号 */
  nickname?: string;
  /** @minLength 6 */
  password: string;
  phone?: string;
}

export interface SKU {
  /** @format int64 */
  id: number;
  name: string;
  /**
   * 兑换所需积分
   * @format int64
   */
  pointsPrice: number;
  soldCount: number;
  stock: number;
}

export interface SKUResponse {
  code: "OK";
  data: SKU;
  message: string;
}

export interface SKUStockInput {
  /**
   * 库存绝对值
   * @min 0
   */
  stock: number;
}

/** 出行方式代码，与 travel_mode 表种子一致 */
export type TravelMode = "walk" | "bike" | "subway" | "bus" | "shared_bike";

export interface TravelModeResponse {
  code: "OK";
  /** 出行方式与排放因子（`travel_mode`） */
  data: AdminTravelMode;
  message: string;
}

/** 至少提交一个字段；缺省字段不修改 */
export interface TravelModeUpdateInput {
  /** @format double */
  efActual?: number;
  /** @format double */
  efBase?: number;
  enabled?: boolean;
  name?: string;
}

export interface Trip {
  cappedByDaily: boolean;
  carbonSavedGrams: number;
  distanceKm: number;
  /** @format int64 */
  grantedPoints: number;
  /** 对外行程 ID（后端内部为自增主键，返回前做了混淆） */
  id: string;
  /** 出行方式代码，与 travel_mode 表种子一致 */
  mode: TravelMode;
  /** @format int64 */
  rawPoints: number;
  /**
   * 行程状态。`POST /api/v1/trips` 目前硬编码返回 `VERIFIED`（历史遗留），
   * 而 `GET /api/v1/trips` 返回库里的 `VALID` / `REVOKED`；前端 service 层
   * 已把 `VERIFIED` 归一成 `VALID`。三个值都列在这里，避免契约与实现打架。
   */
  status: TripStatus;
  /** @format date-time */
  traveledAt: string;
}

export interface TripInput {
  /** 距离区间由 sys_config 配置（种子值 0.1–100 km） */
  distanceKm: number;
  /** 出行方式代码，与 travel_mode 表种子一致 */
  mode: TravelMode;
  /**
   * RFC 3339，必须在服务器当前时间前 7 天内；单日上限按服务器当前自然日计算
   * @format date-time
   */
  traveledAt: string;
}

export interface TripPage {
  items: Trip[];
  page: number;
  pageSize: number;
  /** @format int64 */
  total: number;
}

export interface TripPageResponse {
  code: "OK";
  data: TripPage;
  message: string;
}

export interface TripResponse {
  code: "OK";
  data: Trip;
  message: string;
}

/**
 * 行程状态。`POST /api/v1/trips` 目前硬编码返回 `VERIFIED`（历史遗留），
 * 而 `GET /api/v1/trips` 返回库里的 `VALID` / `REVOKED`；前端 service 层
 * 已把 `VERIFIED` 归一成 `VALID`。三个值都列在这里，避免契约与实现打架。
 */
export type TripStatus = "VALID" | "VERIFIED" | "REVOKED";
