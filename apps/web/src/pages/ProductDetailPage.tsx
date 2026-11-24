import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Badge } from '@astryxdesign/core/Badge'
import { Button } from '@astryxdesign/core/Button'
import { Card } from '@astryxdesign/core/Card'
import { Field } from '@astryxdesign/core/Field'
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
      <Card padding={5} elevation="low">
        <p className="empty-state">商品不存在或已下架。</p>
      </Card>
    )
  }

  const insufficient = (summary?.balance ?? 0) < product.pointsPrice
  const soldOut = product.stock <= 0

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    // Button 的 isDisabled 只落 aria-disabled，回车提交仍会触发表单提交，这里再挡一道
    if (insufficient || soldOut) {
      return
    }
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

  const buttonLabel = insufficient ? '积分不足' : soldOut ? '已兑完' : '确认兑换'

  return (
    <Card padding={5} elevation="low">
      <div className="product-detail">
        <div className="product-emoji product-emoji--large" aria-hidden="true">
          {product.coverUrl ? <img src={product.coverUrl} alt="" /> : productCoverIcon(product.category)}
        </div>
        <div className="product-detail__body">
          <div className="filter-bar">
            {product.category && <span className="chip chip--tiny">{product.category}</span>}
            <Badge
              label={`库存 ${product.stock}`}
              variant={product.stock > 0 ? 'success' : 'neutral'}
            />
            {product.limitPerUser !== undefined && product.limitPerUser !== null && (
              <Badge label={`每人限 ${product.limitPerUser} 件`} variant="info" />
            )}
          </div>
          <h2 className="product-detail__title">{product.name}</h2>
          <p className="muted">{product.description}</p>
          <p className="product-price">{formatPoints(product.pointsPrice)} 分</p>
          <p className="muted">
            我的余额：{formatPoints(summary?.balance ?? 0)} 分
            {insufficient && (
              <span className="danger">
                （还差 {formatPoints(product.pointsPrice - (summary?.balance ?? 0))} 分）
              </span>
            )}
          </p>

          <p className="hint-note">
            兑换成功后积分立即扣减并生成核销码，可在「兑换记录」查看状态；未核销订单支持取消并退回积分。
          </p>

          <form onSubmit={onSubmit} className="product-detail__form">
            <Field label="收货 / 领取人" inputID="receiver">
              <input
                id="receiver"
                className="native-input"
                value={receiver}
                onChange={(e) => setReceiver(e.target.value)}
              />
            </Field>
            {error !== '' && <p className="field-error">{error}</p>}
            <Button
              label={buttonLabel}
              type="submit"
              variant="primary"
              isDisabled={insufficient || soldOut}
            />
          </form>
        </div>
      </div>
    </Card>
  )
}
