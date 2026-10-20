import { useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { cancelOrder, listOrders } from '@/services/orders'
import { ORDER_STATUS_META } from '@/lib/orderStatus'
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

export function OrdersPage(): ReactNode {
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<OrderStatus | 'ALL'>('ALL')
  const [error, setError] = useState('')

  const { data: orders = [] } = useQuery({
    queryKey: ['orders'],
    queryFn: () => Promise.resolve(listOrders()),
  })

  const visible = filter === 'ALL' ? orders : orders.filter((o) => o.status === filter)

  const onCancel = (id: string) => {
    setError('')
    try {
      cancelOrder(id)
      void queryClient.invalidateQueries({ queryKey: ['orders'] })
      void queryClient.invalidateQueries({ queryKey: ['summary'] })
      void queryClient.invalidateQueries({ queryKey: ['txns'] })
    } catch (err) {
      setError(err instanceof Error ? err.message : '取消失败')
    }
  }

  return (
    <>
      <section className="hero">
        <h2>兑换记录</h2>
        <p>订单状态机：待支付 → 已支付 → 已发货 → 已完成；仅待支付可取消并返还积分。</p>
      </section>

      <div className="filter-bar">
        {FILTERS.map((f) => (
          <button
            key={f.value}
            className={filter === f.value ? 'chip chip--active' : 'chip'}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && <p className="field-error">{error}</p>}

      <section className="card">
        {visible.length === 0 ? (
          <p className="empty-state">该状态下暂无订单。</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>订单号</th>
                <th>商品</th>
                <th>消耗积分</th>
                <th>状态</th>
                <th>下单时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((o) => {
                const meta = ORDER_STATUS_META[o.status]
                return (
                  <tr key={o.id}>
                    <td><code>{o.id}</code></td>
                    <td>{o.productName}</td>
                    <td>-{formatPoints(o.pointsSpent)}</td>
                    <td>
                      <span className={`badge ${meta.badge}`}>{meta.label}</span>
                    </td>
                    <td>{formatDateTime(o.createdAt)}</td>
                    <td>
                      {meta.cancellable && (
                        <button className="btn btn--danger" onClick={() => onCancel(o.id)}>
                          取消并返还
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </section>
    </>
  )
}
