import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { listProducts } from '@/services/products'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'
import type { ReactNode } from 'react'

export function MallPage(): ReactNode {
  const { data: products = [] } = useQuery({
    queryKey: ['products'],
    queryFn: () => Promise.resolve(listProducts()),
  })
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => Promise.resolve(getSummary()),
  })

  return (
    <>
      <section className="hero">
        <h2>积分商城</h2>
        <p>用积攒的碳积分兑换绿色好物；兑换扣减以后端事务为准，原型仅做预校验。</p>
      </section>
      <p className="muted">当前余额：{formatPoints(summary?.balance ?? 0)} 分</p>
      <div className="grid product-grid">
        {products.map((p) => (
          <div key={p.id} className="card product-card">
            <div className="product-emoji" aria-hidden="true">
              {p.imageEmoji}
            </div>
            <h3>{p.name}</h3>
            <p className="muted">{p.description}</p>
            <p className="product-price">{formatPoints(p.pointsPrice)} 分</p>
            <p className="muted">库存 {p.stock}</p>
            <Link to={`/mall/${p.id}`} className="btn btn--ghost">
              查看详情
            </Link>
          </div>
        ))}
      </div>
    </>
  )
}
