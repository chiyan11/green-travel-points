import { useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listAdminOrders,
  transitionAdminOrder,
  type AdminOrderAction,
} from '@/services/admin'
import { ORDER_STATUS_META } from '@/lib/orderStatus'
import { formatDateTime, formatPoints } from '@/lib/format'
import { ApiError } from '@/lib/http'
import type { AdminOrder } from '@/types/admin'
import type { OrderStatus } from '@/types/domain'

const FILTERS: Array<{ value: OrderStatus | ''; label: string }> = [
  { value: '', label: '全部' },
  { value: 'PENDING', label: '待支付' },
  { value: 'PAID', label: '已支付' },
  { value: 'SHIPPED', label: '已发货' },
  { value: 'COMPLETED', label: '已完成' },
  { value: 'CANCELLED', label: '已取消' },
]

/** 按当前状态给出下一步动作：PENDING→确认收款、PAID→发货、SHIPPED→核销完成。 */
const NEXT_ACTION: Partial<Record<OrderStatus, { action: AdminOrderAction; label: string }>> = {
  PENDING: { action: 'confirm', label: '确认收款' },
  PAID: { action: 'ship', label: '发货' },
  SHIPPED: { action: 'complete', label: '核销完成' },
}

export function AdminOrdersPage(): ReactNode {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<OrderStatus | ''>('')
  const [error, setError] = useState('')

  const { data, isPending, isError, error: queryError } = useQuery({
    queryKey: ['admin-orders', status],
    queryFn: () => listAdminOrders(status),
  })

  const mutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: AdminOrderAction }) =>
      transitionAdminOrder(id, action),
    onSuccess: () => {
      setError('')
      void queryClient.invalidateQueries({ queryKey: ['admin-orders'] })
    },
    onError: (err: unknown) => {
      setError(err instanceof ApiError ? err.message : '状态流转失败')
    },
  })

  const orders = data?.items ?? []

  return (
    <>
      <div className="filter-bar">
        {FILTERS.map((f) => (
          <button
            key={f.value || 'ALL'}
            className={status === f.value ? 'chip chip--active' : 'chip'}
            onClick={() => setStatus(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && <p className="field-error">{error}</p>}

      <section className="card">
        <div className="section-head">
          <h3>兑换订单</h3>
          <span className="muted">共 {data?.total ?? 0} 单</span>
        </div>

        {isPending ? (
          <p className="empty-state">加载中…</p>
        ) : isError ? (
          <p className="field-error">
            {queryError instanceof ApiError && queryError.status === 403
              ? '当前账号没有管理员权限，请用 ADMIN 账号登录。'
              : queryError instanceof Error
                ? queryError.message
                : '加载失败'}
          </p>
        ) : orders.length === 0 ? (
          <p className="empty-state">该状态下暂无订单。</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>订单号</th>
                <th>用户</th>
                <th>商品</th>
                <th>积分</th>
                <th>收货人</th>
                <th>状态</th>
                <th>下单时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o: AdminOrder) => {
                const meta = ORDER_STATUS_META[o.status]
                const next = NEXT_ACTION[o.status]
                return (
                  <tr key={o.id}>
                    <td><code>{o.orderNo}</code></td>
                    <td>{o.userId}</td>
                    <td>
                      {o.productName}（{o.skuName}）× {o.quantity}
                    </td>
                    <td>-{formatPoints(o.pointsSpent)}</td>
                    <td>{o.receiver}</td>
                    <td>
                      <span className={`badge ${meta.badge}`}>{meta.label}</span>
                    </td>
                    <td>{formatDateTime(o.createdAt)}</td>
                    <td>
                      {next ? (
                        <button
                          className="btn btn--ghost"
                          disabled={mutation.isPending}
                          onClick={() => mutation.mutate({ id: o.id, action: next.action })}
                        >
                          {next.label}
                        </button>
                      ) : (
                        <span className="muted">—</span>
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
