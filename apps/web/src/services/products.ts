/**
 * 商品服务。
 * GET /api/v1/products（公开）、GET /api/v1/products/{id}（公开）。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成），本文件只做
 * wire 契约 → 页面视图模型（types/domain.ts）的映射，不再手写后端字段形状。
 */
import { api } from '@/lib/http'
import type { Product as WireProduct, ProductPage } from '@/types/api'
import type { Product, ProductStatus } from '@/types/domain'

/**
 * `status` 为 omitempty 字段：缺失时回退到历史字段 `active`；
 * 出现了前端不认识的取值时按「下架」处理，绝不能默认成可兑换的 ONLINE。
 */
function normalizeStatus(r: WireProduct): ProductStatus {
  if (r.status === 'ONLINE' || r.status === 'OFFLINE' || r.status === 'DRAFT') return r.status
  if (typeof r.status === 'string' && r.status !== '') return 'OFFLINE'
  return r.active ? 'ONLINE' : 'OFFLINE'
}

function toProduct(r: WireProduct): Product {
  return {
    id: String(r.id),
    name: r.name,
    description: r.description,
    category: r.category,
    coverUrl: r.imageUrl,
    status: normalizeStatus(r),
    pointsPrice: r.pointsPrice,
    stock: r.stock,
    limitPerUser: r.limitPerUser,
  }
}

export interface ProductList {
  items: Product[]
  page: number
  pageSize: number
  total: number
}

export async function listProducts(page = 1, pageSize = 50): Promise<ProductList> {
  const res = await api.get<ProductPage>(`/api/v1/products?page=${page}&pageSize=${pageSize}`)
  return {
    // 下架商品后端也会返回，这里只留可兑换的
    items: (res.items ?? []).map(toProduct).filter((p) => p.status === 'ONLINE'),
    page: res.page,
    pageSize: res.pageSize,
    total: res.total,
  }
}

/**
 * 单个商品。失败（404 / 500 / 网络）一律抛给调用方：页面要能区分
 * 「商品不存在」和「加载失败」，不能都画成同一句提示。
 */
export async function getProduct(id: string): Promise<Product> {
  const raw = await api.get<WireProduct>(`/api/v1/products/${encodeURIComponent(id)}`)
  return toProduct(raw)
}
