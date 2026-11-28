/**
 * 运营后台（M8）类型：只服务 /admin 页面。
 *
 * 与消费者端 types/domain.ts 分开，避免后台才有的字段（skuId、userId、售出数）
 * 漏进消费者端类型。字段口径对齐 apps/api/internal/app/types.go 的 JSON tag。
 */
import type { OrderStatus, ProductStatus } from '@/types/domain'

/** 后台 SKU：id 用来调 PUT /api/v1/admin/skus/{id}/stock。 */
export interface AdminSKU {
  id: string
  name: string
  pointsPrice: number
  stock: number
  soldCount: number
}

/** 后台商品：列表接口带 skus 明细，可直接改库存。 */
export interface AdminProduct {
  id: string
  merchantId: string
  name: string
  description: string
  category: string
  coverUrl: string
  status: ProductStatus
  pointsPrice: number
  stock: number
  limitPerUser: number
  skus: AdminSKU[]
}

/** 新建商品入参：merchantId 由页面固定填（当前只有一家商户）。 */
export interface AdminProductInput {
  merchantId: number
  name: string
  description: string
  category: string
  status: ProductStatus
  limitPerUser: number
  skus: Array<{ name: string; pointsPrice: number; stock: number }>
}

/** 后台订单：全部用户的兑换单。 */
export interface AdminOrder {
  id: string
  userId: string
  orderNo: string
  productName: string
  skuName: string
  pointsSpent: number
  quantity: number
  receiver: string
  status: OrderStatus
  /** 后端原始 status，UNKNOWN 时用于展示。 */
  statusText: string
  redeemCode: string
  createdAt: string
}

export const PRODUCT_STATUS_META: Record<ProductStatus, { label: string; badge: string }> = {
  DRAFT: { label: '草稿', badge: 'badge--muted' },
  ONLINE: { label: '已上架', badge: 'badge--ok' },
  OFFLINE: { label: '已下架', badge: 'badge--warn' },
}
