/**
 * 运营后台服务（M8）：只被 /admin 页面调用。
 *
 * 对接 apps/api/internal/httpserver/admin.go 的 8 个端点：
 *   GET    /api/v1/admin/products?status=&keyword=&page=&pageSize=
 *   POST   /api/v1/admin/products
 *   PATCH  /api/v1/admin/products/{id}
 *   PUT    /api/v1/admin/skus/{id}/stock
 *   GET    /api/v1/admin/orders?status=&page=&pageSize=
 *   POST   /api/v1/admin/orders/{id}/(confirm|ship|complete)
 *
 * 后端原始 JSON 用 number 主键与 imageUrl 字段，这里统一收成 string 与 coverUrl，
 * 页面只看干净的后台类型。
 */
import { api } from '@/lib/http'
import type { Page } from '@/types/domain'
import type { AdminOrder, AdminProduct, AdminProductInput } from '@/types/admin'
import { toOrderStatus } from '@/services/orders'

interface RawSKU {
  id: number
  name: string
  pointsPrice: number
  stock: number
  soldCount?: number
}

interface RawAdminProduct {
  id: number
  merchantId?: number
  name: string
  description?: string
  category?: string
  imageUrl?: string
  status?: string
  pointsPrice: number
  stock: number
  limitPerUser?: number
  skus?: RawSKU[]
}

interface RawAdminOrder {
  id: number
  userId: number
  orderNo: string
  productName: string
  skuName: string
  pointsSpent: number
  quantity: number
  receiver: string
  status: string
  redeemCode?: string
  createdAt: string
}

function toAdminProduct(r: RawAdminProduct): AdminProduct {
  const status =
    r.status === 'ONLINE' || r.status === 'DRAFT' || r.status === 'OFFLINE' ? r.status : 'OFFLINE'
  return {
    id: String(r.id),
    merchantId: String(r.merchantId ?? 0),
    name: r.name,
    description: r.description ?? '',
    category: r.category ?? '',
    coverUrl: r.imageUrl ?? '',
    status,
    pointsPrice: r.pointsPrice,
    stock: r.stock,
    limitPerUser: r.limitPerUser ?? 0,
    skus: (r.skus ?? []).map((s) => ({
      id: String(s.id),
      name: s.name,
      pointsPrice: s.pointsPrice,
      stock: s.stock,
      soldCount: s.soldCount ?? 0,
    })),
  }
}

function toAdminOrder(r: RawAdminOrder): AdminOrder {
  return {
    id: String(r.id),
    userId: String(r.userId),
    orderNo: r.orderNo,
    productName: r.productName,
    skuName: r.skuName,
    pointsSpent: r.pointsSpent,
    quantity: r.quantity,
    receiver: r.receiver,
    status: toOrderStatus(r.status),
    statusText: r.status ?? '',
    redeemCode: r.redeemCode ?? '',
    createdAt: r.createdAt,
  }
}

export interface AdminProductQuery {
  status?: string
  keyword?: string
  page?: number
  pageSize?: number
}

export async function listAdminProducts(q: AdminProductQuery = {}): Promise<Page<AdminProduct>> {
  const params = new URLSearchParams()
  params.set('page', String(q.page ?? 1))
  params.set('pageSize', String(q.pageSize ?? 20))
  if (q.status) params.set('status', q.status)
  if (q.keyword) params.set('keyword', q.keyword)

  const page = await api.get<Page<RawAdminProduct>>(`/api/v1/admin/products?${params.toString()}`)
  return {
    items: (page.items ?? []).map(toAdminProduct),
    page: page.page,
    pageSize: page.pageSize,
    total: page.total,
  }
}

export async function createAdminProduct(input: AdminProductInput): Promise<AdminProduct> {
  const raw = await api.post<RawAdminProduct>('/api/v1/admin/products', input)
  return toAdminProduct(raw)
}

/** 上下架：只发 status，其余字段不动。 */
export async function updateAdminProductStatus(
  id: string,
  status: AdminProduct['status'],
): Promise<AdminProduct> {
  const raw = await api.patch<RawAdminProduct>(`/api/v1/admin/products/${encodeURIComponent(id)}`, {
    status,
  })
  return toAdminProduct(raw)
}

export async function setSKUStock(skuId: string, stock: number): Promise<void> {
  await api.put<RawSKU>(`/api/v1/admin/skus/${encodeURIComponent(skuId)}/stock`, { stock })
}

export async function listAdminOrders(
  status: string,
  page = 1,
  pageSize = 20,
): Promise<Page<AdminOrder>> {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
  if (status) params.set('status', status)
  const res = await api.get<Page<RawAdminOrder>>(`/api/v1/admin/orders?${params.toString()}`)
  return {
    items: (res.items ?? []).map(toAdminOrder),
    page: res.page,
    pageSize: res.pageSize,
    total: res.total,
  }
}

export type AdminOrderAction = 'confirm' | 'ship' | 'complete'

/** 订单流转：confirm(PENDING→PAID) / ship(PAID→SHIPPED) / complete(SHIPPED→COMPLETED)。 */
export async function transitionAdminOrder(
  id: string,
  action: AdminOrderAction,
): Promise<AdminOrder> {
  const raw = await api.post<RawAdminOrder>(
    `/api/v1/admin/orders/${encodeURIComponent(id)}/${action}`,
    {},
  )
  return toAdminOrder(raw)
}
