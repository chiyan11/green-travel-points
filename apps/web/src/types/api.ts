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
   * RFC 3339，决定归属哪一天的单日上限
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

export type TripStatus = "VALID" | "REVOKED";
