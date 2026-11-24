import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { listProducts } from '@/services/products'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'
import { productCoverIcon } from '@/lib/productCover'
import type { ReactNode } from 'react'

export function MallPage(): ReactNode {
  const { data: products = [] } = useQuery({
    queryKey: ['products'],
    queryFn: () => listProducts(),
  })
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => getSummary(),
  })

  return (
    <>
      <section className="hero">
        <h2>积分商城</h2>
        <p>用积攒的碳积分兑换绿色好物；兑换扣减以后端事务为准，原型仅做预校验。</p>
      </section>

      <div className="row-between">
        <span className="muted" style={{ margin: 0 }}>共 {products.length} 件在售商品</span>
        <span className="price-chip">我的余额 {formatPoints(summary?.balance ?? 0)} 分</span>
      </div>

      {products.length === 0 ? (
        <p className="empty-state">暂无在售商品，等上架后再来看看。</p>
      ) : (
        <div className="grid product-grid">
          {products.map((p) => (
            <div key={p.id} className="card product-card">
              <div className="product-cover" aria-hidden="true">
                {p.coverUrl ? <img src={p.coverUrl} alt="" /> : productCoverIcon(p.category)}
              </div>
              <div className="product-body">
                {p.category && <span className="chip chip--tiny">{p.category}</span>}
                <h3>{p.name}</h3>
                <p className="muted" style={{ margin: 0 }}>{p.description}</p>
                <div className="product-meta">
                  <span className="price-chip">{formatPoints(p.pointsPrice)} 分</span>
                  <span className="muted" style={{ margin: 0 }}>库存 {p.stock}</span>
                </div>
                <Link to={`/mall/${p.id}`} className="btn btn--ghost">
                  查看详情
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  )
}
