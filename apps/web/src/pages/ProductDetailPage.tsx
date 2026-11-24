import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { IconGift, IconPackage, IconTicket } from '@/components/icons'
import { getProduct } from '@/services/products'
import { createOrder } from '@/services/orders'
import { getSummary } from '@/services/points'
import { formatPoints } from '@/lib/format'
import { productCoverIcon } from '@/lib/productCover'

/** 商品详情：价格与「兑换后还剩多少分」是决策信息，放在同一屏内。 */
export function ProductDetailPage(): ReactNode {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [receiver, setReceiver] = useState('本人')
  const [error, setError] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [pending, setPending] = useState(false)

  const { data: product } = useQuery({
    queryKey: ['product', id],
    queryFn: async () => (id ? getProduct(id) : undefined),
  })
  const { data: summary } = useQuery({ queryKey: ['summary'], queryFn: () => getSummary() })

  if (!product) {
    return (
      <div className="page">
        <div className="card">
          <p className="empty-state">商品不存在或已下架。</p>
        </div>
      </div>
    )
  }

  const balance = summary?.balance ?? 0
  const insufficient = balance < product.pointsPrice
  const soldOut = product.stock <= 0
  const shortfall = product.pointsPrice - balance

  const redeem = async () => {
    if (insufficient || soldOut) return
    setError('')
    setPending(true)
    try {
      const order = await createOrder(product.id, receiver)
      // 先等余额重新取回来再跳转：顶栏余额和兑换页读的是同一个 summary，
      // 直接 void 掉会在跳转后才刷新，用户会先看到兑换前的余额。
      await queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      navigate(`/orders?created=${order.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : '兑换失败')
      setConfirming(false)
    } finally {
      setPending(false)
    }
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    // 禁用态只挡点击，回车仍会提交表单，这里再挡一道
    if (insufficient || soldOut) return
    setConfirming(true)
  }

  const buttonLabel = insufficient ? '积分不足' : soldOut ? '已兑完' : '确认兑换'

  const cover = product.coverUrl ? (
    <img src={product.coverUrl} alt="" />
  ) : productCoverIcon(product.category ?? '') === '🚇' ? (
    <IconTicket size={96} />
  ) : productCoverIcon(product.category ?? '') === '📓' ? (
    <IconPackage size={96} />
  ) : (
    <IconGift size={96} />
  )

  return (
    <div className="page">
      <div className="page__head">
        <Link className="btn btn--ghost" to="/mall">
          ← 返回商城
        </Link>
      </div>

      <section className="card">
        <div className="detail">
          <div className="detail__cover" aria-hidden="true">
            {cover}
          </div>

          <div className="detail__body">
            <div className="detail__tags">
              {product.category && <span className="tag">{product.category}</span>}
              <span className={product.stock > 0 ? 'badge badge--ok' : 'badge badge--muted'}>
                库存 {product.stock}
              </span>
              {product.limitPerUser ? (
                <span className="badge badge--info">每人限 {product.limitPerUser} 件</span>
              ) : null}
            </div>

            <h1 className="detail__title">{product.name}</h1>
            <p className="detail__desc">{product.description}</p>

            <p className="detail__price">
              <span className="detail__price-value">{formatPoints(product.pointsPrice)}</span>
              <span className="detail__price-unit">分</span>
            </p>
            <p className="detail__balance">
              {insufficient ? (
                <>
                  当前余额 {formatPoints(balance)} 分，兑换需再攒{' '}
                  <span className="shortfall">{formatPoints(shortfall)} 分</span>
                </>
              ) : (
                <>兑换后剩余 {formatPoints(balance - product.pointsPrice)} 分</>
              )}
            </p>

            <form className="detail__form" onSubmit={onSubmit}>
              <label className="field" htmlFor="receiver">
                <span className="field__label">收货 / 领取人</span>
                <input
                  id="receiver"
                  className="native-input"
                  value={receiver}
                  onChange={(e) => setReceiver(e.target.value)}
                />
              </label>
              {error !== '' && <p className="field-error">{error}</p>}
              <button
                type="submit"
                className="btn btn--primary btn--block"
                disabled={insufficient || soldOut}
                aria-disabled={insufficient || soldOut}
              >
                {buttonLabel}
              </button>
            </form>
          </div>
        </div>
      </section>

      {confirming && (
        <div className="modal" role="dialog" aria-modal="true" aria-labelledby="redeem-title">
          <div className="modal__panel">
            <h2 className="modal__title" id="redeem-title">
              确认用 {formatPoints(product.pointsPrice)} 分兑换？
            </h2>
            <p className="modal__note">
              兑换成功后积分立即扣减，并生成一个线下核销码；在「兑换记录」里可以查看状态，
              未核销的订单支持取消并退回积分。
            </p>
            <dl className="modal__rows">
              <div>
                <dt>商品</dt>
                <dd>{product.name}</dd>
              </div>
              <div>
                <dt>领取人</dt>
                <dd>{receiver || '本人'}</dd>
              </div>
              <div>
                <dt>兑换后余额</dt>
                <dd>{formatPoints(balance - product.pointsPrice)} 分</dd>
              </div>
            </dl>
            <div className="modal__actions">
              <button
                type="button"
                className="btn btn--ghost"
                onClick={() => setConfirming(false)}
                disabled={pending}
              >
                再想想
              </button>
              <button
                type="button"
                className="btn btn--primary"
                onClick={() => void redeem()}
                disabled={pending}
              >
                {pending ? '兑换中…' : '确认兑换并生成核销码'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
