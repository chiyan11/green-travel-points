import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getProduct } from '@/services/products'
import { createOrder } from '@/services/orders'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'

export function ProductDetailPage(): ReactNode {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [receiver, setReceiver] = useState('本人')
  const [error, setError] = useState('')

  const { data: product } = useQuery({
    queryKey: ['product', id],
    queryFn: () => Promise.resolve(id ? getProduct(id) : undefined),
  })
  const { data: summary } = useQuery({
    queryKey: ['summary'],
    queryFn: () => Promise.resolve(getSummary()),
  })

  if (!product) {
    return (
      <section className="card">
        <p className="empty-state">商品不存在或已下架。</p>
      </section>
    )
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const order = createOrder(product.id, receiver)
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
          {product.imageEmoji}
        </div>
        <div>
          <h2>{product.name}</h2>
          <p className="muted">{product.description}</p>
          <p className="product-price">{formatPoints(product.pointsPrice)} 分</p>
          <p className="muted">库存：{product.stock} · 分类：{product.category}</p>
          {product.limitPerUser && (
            <p className="muted">每人限兑 {product.limitPerUser} 件</p>
          )}
          <p className="muted">我的余额：{formatPoints(summary?.balance ?? 0)} 分</p>
        </div>
      </div>

      <form onSubmit={onSubmit}>
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
