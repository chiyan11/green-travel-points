import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { IconGift, IconPackage, IconTicket } from '@/components/icons'
import { listProducts } from '@/services/products'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'
import { productCoverKind } from '@/lib/productCover'
import type { Product } from '@/types/domain'

/** 按每天能攒的积分粗估「要攒几天」，让价格有可负担性的参照。 */
const POINTS_PER_DAY_HINT = 75

function coverIcon(category: string | undefined, coverUrl: string | undefined): ReactNode {
  if (coverUrl) return <img src={coverUrl} alt="" />
  switch (productCoverKind(category)) {
    case 'ticket':
      return <IconTicket size={64} />
    case 'package':
      return <IconPackage size={64} />
    default:
      return <IconGift size={64} />
  }
}

/** 积分商城：买得起才能换，买不起也要让用户知道还差多少。 */
export function MallPage(): ReactNode {
  const { data: products = [] } = useQuery({ queryKey: ['products'], queryFn: () => listProducts() })
  const { data: summary } = useQuery({ queryKey: ['summary'], queryFn: () => getSummary() })
  const balance = summary?.balance ?? 0

  return (
    <div className="page">
      <div className="page__head">
        <div>
          <h1 className="page-title">积分商城</h1>
          <p className="page-sub">用碳积分换绿色好物，兑换扣减以后端为准。</p>
        </div>
        <span className="balance-chip">我的余额 {formatPoints(balance)} 分</span>
      </div>

      <div className="section-head">
        <span className="muted">共 {products.length} 件在售商品</span>
      </div>

      {products.length === 0 ? (
        <div className="empty">
          <span className="empty__art">
            <IconGift size={48} />
          </span>
          <p className="empty__title">暂无在售商品，等上架后再来看看。</p>
        </div>
      ) : (
        <div className="product-grid">
          {products.map((p: Product) => {
            const short = balance < p.pointsPrice
            const days = Math.max(1, Math.ceil(p.pointsPrice / POINTS_PER_DAY_HINT))
            return (
              <article key={p.id} className={short ? 'product-card is-unaffordable' : 'product-card'}>
                <div className="product-cover" aria-hidden="true">
                  {coverIcon(p.category, p.coverUrl)}
                </div>
                <div className="product-body">
                  {p.category && <span className="tag">{p.category}</span>}
                  <h3 className="product-name">{p.name}</h3>
                  <p className="product-desc">{p.description}</p>
                  <div className="price-row">
                    <span className="price">{formatPoints(p.pointsPrice)} 分</span>
                    <span className="price-hint">≈ 攒 {days} 天</span>
                    <span className={p.stock <= 5 ? 'stock stock--low' : 'stock'}>库存 {p.stock}</span>
                  </div>
                  <Link className="btn" to={`/mall/${p.id}`}>
                    {short ? `还差 ${formatPoints(p.pointsPrice - balance)} 分` : '查看详情'}
                  </Link>
                </div>
              </article>
            )
          })}
        </div>
      )}
    </div>
  )
}
