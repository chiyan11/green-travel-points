/**
 * 商品服务（联调版）。
 * GET /api/v1/products（公开）、GET /api/v1/products/{id}（公开）。
 * 字段做双向兼容：P3 当前返回 imageUrl/active，迁移口径为 coverUrl/status，两者都能解析。
 */
import { api } from '@/lib/http'
import type { Page, Product, ProductStatus } from '@/types/domain'

/** 后端原始 DTO（P3 实现，含过渡期别名）。 */
interface RawProduct {
  id: number
  name: string
  description?: string
  category?: string
  imageUrl?: string
  coverUrl?: string
  status?: string
  active?: boolean
  pointsPrice: number
  stock: number
  limitPerUser?: number
}

function normalizeStatus(r: RawProduct): ProductStatus {
  if (r.status === 'ONLINE' || r.status === 'OFFLINE' || r.status === 'DRAFT') return r.status
  return r.active ? 'ONLINE' : 'OFFLINE'
}

function toProduct(r: RawProduct): Product {
  return {
    id: String(r.id),
    name: r.name,
    description: r.description ?? '',
    category: r.category ?? '',
    coverUrl: r.coverUrl || r.imageUrl || '',
    status: normalizeStatus(r),
    pointsPrice: r.pointsPrice,
    stock: r.stock,
    limitPerUser: r.limitPerUser ?? 0,
  }
}

export async function listProducts(): Promise<Product[]> {
  const page = await api.get<Page<RawProduct>>('/api/v1/products?page=1&pageSize=50')
  return (page.items ?? []).map(toProduct).filter((p) => p.status === 'ONLINE')
}

export async function getProduct(id: string): Promise<Product | undefined> {
  try {
    const raw = await api.get<RawProduct>(`/api/v1/products/${encodeURIComponent(id)}`)
    return toProduct(raw)
  } catch {
    return undefined
  }
}
