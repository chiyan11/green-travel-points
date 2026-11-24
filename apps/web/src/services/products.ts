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

/** `status` 为 omitempty 字段，缺失时回退到历史字段 `active`。 */
function normalizeStatus(r: WireProduct): ProductStatus {
  if (r.status === 'ONLINE' || r.status === 'OFFLINE' || r.status === 'DRAFT') return r.status
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

export async function listProducts(): Promise<Product[]> {
  const page = await api.get<ProductPage>('/api/v1/products?page=1&pageSize=50')
  return (page.items ?? []).map(toProduct).filter((p) => p.status === 'ONLINE')
}

export async function getProduct(id: string): Promise<Product | undefined> {
  try {
    const raw = await api.get<WireProduct>(`/api/v1/products/${encodeURIComponent(id)}`)
    return toProduct(raw)
  } catch {
    return undefined
  }
}
