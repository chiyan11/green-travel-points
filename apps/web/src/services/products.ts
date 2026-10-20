/**
 * 商品服务（MOCK）。
 * TODO 联调替换：GET /api/v1/products（分页+分类）、GET /api/v1/products/{id}。
 */
import { mockDb } from '@/mock/data'
import type { Product } from '@/types/domain'

export function listProducts(): Product[] {
  return mockDb.getProducts().filter((p) => p.active)
}

export function getProduct(id: string): Product | undefined {
  return mockDb.getProduct(id)
}
