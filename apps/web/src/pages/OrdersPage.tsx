import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { IconEmptyRoute } from '@/components/icons'
import { cancelOrder, listOrders } from '@/services/orders'
import { ORDER_STATUS_META, orderStatusBadgeClass } from '@/lib/orderStatus'
import { formatDateTime, formatPoints } from '@/lib/format'
import type { OrderStatus } from '@/types/domain'

const FILTERS: Array<{ value: OrderStatus | 'ALL'; label: string }> = [
  { value: 'ALL', label: '全部' },
  { value: 'PENDING', label: '待支付' },
  { value: 'PAID', label: '已支付' },
  { value: 'SHIPPED', label: '已发货' },
  { value: 'COMPLETED', label: '已完成' },
  { value: 'CANCELLED', label: '已取消' },
]

/**
 * 兑换记录：卡片行而不是表格。
 * 核销码是用户线下要出示的东西，所以放在最左边、等宽字体放大；订单号降到小字。
 */
export function OrdersPage(): ReactNode {
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<OrderStatus | 'ALL'>('ALL')
  const [error, setError] = useState('')

  const { data: orders = [], isPending: ordersPending } = useQuery({
    queryKey: ['orders'],
    queryFn: () => listOrders(),
  })
  const visible = filter === 'ALL' ? orders : orders.filter((o) => o.status === filter)

  const onCancel = async (id: string) => {
    setError('')
    try {
      await cancelOrder(id)
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      void queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['txns'] })
    } catch (err) {
      setError(err instanceof Error ? err.message : '取消失败')
    }
  }

  return (
    <div className="page">
      <div className="page__head">
        <div>
          <h1 className="page-title">兑换记录</h1>
          <p className="page-sub">未核销的订单可以取消，积分会原路退回。</p>
        </div>
      </div>

      <div className="segmented" role="group" aria-label="按状态筛选">
        {FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            className={filter === f.value ? 'segmented__btn is-active' : 'segmented__btn'}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error !== '' && <p className="field-error">{error}</p>}

      <section className="card">
        <div className="card__head">
          <h2 className="card__title">
            {filter === 'ALL' ? '全部订单' : `${FILTERS.find((f) => f.value === filter)?.label ?? ''}订单`}
          </h2>
          <span className="muted">{ordersPending ? '加载中…' : `共 ${visible.length} 条`}</span>
        </div>

        {ordersPending ? (
          <div className="order-list">
            {[0, 1].map((i) => (
              <div className="order-card" key={i}>
                <div>
                  <span className="skeleton skeleton--line" style={{ width: 60 }} />
                  <span className="skeleton skeleton--line" style={{ width: 160, height: 18, marginTop: 8 }} />
                </div>
                <div>
                  <span className="skeleton skeleton--line" style={{ width: 120 }} />
                  <span className="skeleton skeleton--line" style={{ width: 180, marginTop: 8 }} />
                </div>
              </div>
            ))}
          </div>
        ) : visible.length === 0 ? (
          <div className="empty">
            <span className="empty__art">
              <IconEmptyRoute size={44} />
            </span>
            <p className="empty__title">该状态下暂无订单。</p>
            <p className="empty__note">去商城看看，攒下的积分可以换点实用的东西。</p>
          </div>
        ) : (
          <div className="order-list">
            {visible.map((o) => {
              const meta = ORDER_STATUS_META[o.status]
              return (
                <article className="order-card" key={o.id}>
                  <div>
                    <p className="order-code__label">核销码</p>
                    <p className="order-code">{o.redeemCode || '—'}</p>
                  </div>

                  <div>
                    <p className="order-main__title">{o.productName}</p>
                    <p className="order-main__meta">
                      订单 {o.orderNo} · {formatDateTime(o.createdAt)}
                    </p>
                  </div>

                  <div className="order-side">
                    <span className={`badge ${orderStatusBadgeClass(o.status)}`}>{meta.label}</span>
                    <span className="muted">-{formatPoints(o.pointsSpent)} 分</span>
                  </div>

                  <div className="order-actions">
                    {meta.cancellable ? (
                      <button
                        type="button"
                        className="btn btn--inline"
                        onClick={() => void onCancel(o.id)}
                      >
                        取消并返还
                      </button>
                    ) : o.status === 'CANCELLED' ? (
                      <Link className="btn btn--ghost btn--inline" to="/mall">
                        重新兑换
                      </Link>
                    ) : null}
                  </div>
                </article>
              )
            })}
          </div>
        )}
      </section>
    </div>
  )
}
