import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getProduct } from '@/services/products'
import { createOrder } from '@/services/orders'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'
import { productCoverIcon } from '@/lib/productCover'

export function ProductDetailPage(): ReactNode {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [receiver, setReceiver] = useState('本人')
  const [error, setError] = useState('')

  const { data: product } = useQuery({
    queryKey: ['product', id],
    queryFn: async () => (id ? getProduct(id) : undefined),
  })
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => getSummary(),
  })

  if (!product) {
    return (
      <section className="card">
        <p className="empty-state">商品不存在或已下架。</p>
      </section>
    )
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const order = await createOrder(product.id, receiver)
      void queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      navigate(`/orders?created=${order.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : '兑换失败')
    }
  }

  const insufficient = (summary?.balance ?? 0) < product.pointsPrice

  return (
    <section className="card">
      <div className="product-detail">
        <div className="product-emoji product-emoji--large" aria-hidden="true">
          {product.coverUrl ? <img src={product.coverUrl} alt="" /> : productCoverIcon(product.category)}
        </div>
        <div style={{ flex: 1, minWidth: 240 }}>
          <div className="filter-bar" style={{ marginBottom: 10 }}>
            {product.category && <span className="chip chip--tiny">{product.category}</span>}
            <span className={product.stock > 0 ? 'badge badge--ok' : 'badge badge--muted'}>
              库存 {product.stock}
            </span>
            {product.limitPerUser && (
              <span className="badge badge--info">每人限 {product.limitPerUser} 件</span>
            )}
          </div>
          <h2 style={{ margin: '0 0 6px', fontSize: 24 }}>{product.name}</h2>
          <p className="muted" style={{ margin: 0 }}>{product.description}</p>
          <p className="product-price">{formatPoints(product.pointsPrice)} 分</p>
          <p className="muted" style={{ margin: 0 }}>
            我的余额：{formatPoints(summary?.balance ?? 0)} 分
            {insufficient && <span className="danger">（还差 {formatPoints(product.pointsPrice - (summary?.balance ?? 0))} 分）</span>}
          </p>
        </div>
      </div>

      <div className="hint-note">
        兑换成功后积分立即扣减并生成核销码，可在「兑换记录」查看状态；未核销订单支持取消并退回积分。
      </div>

      <form onSubmit={onSubmit} style={{ marginTop: 18 }}>
        <div className="form-field">
          <label htmlFor="receiver">收货 / 领取人</label>
          <input
            id="receiver"
            value={receiver}
            onChange={(e) => setReceiver(e.target.value)}
          />
        </div>
        {error && <p className="field-error">{error}</p>}
        <button type="submit" className="btn" disabled={insufficient || product.stock <= 0}>
          {insufficient ? '积分不足' : product.stock <= 0 ? '已兑完' : '确认兑换'}
        </button>
      </form>
    </section>
  )
}
