import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { listAdminProducts } from '@/services/admin'
import { ApiError } from '@/lib/http'

/**
 * 运营后台的角色门（N22）。
 *
 * 后端每个 /admin 端点都挂了 RequireAdmin，但登录响应与 /me 都不带 role，
 * 前端在路由层无法直接判断。这里用一次最小探测代替：
 * `GET /admin/products?page=1&pageSize=1` 返回 200 = 管理员，403 = 普通市民。
 *
 * - `retry: false`：403 是确定性结果，重试只会拖慢无权限面板的出现；
 * - `staleTime: 60s`：进入后台三屏时只探测一次，切页不重复打接口；
 * - 探测失败（网络/500）不会放行，显示错误面板并给回首页入口。
 *
 * 长期方案是给 `/me` 与登录响应补 `role`（见 N14 契约），届时这里可退化为读会话。
 */
const ADMIN_PROBE_KEY = ['admin-access'] as const

export function AdminGate({ children }: { children: ReactNode }) {
  const { isPending, isError, error } = useQuery({
    queryKey: ADMIN_PROBE_KEY,
    queryFn: () => listAdminProducts({ page: 1, pageSize: 1 }),
    retry: false,
    staleTime: 60_000,
  })

  if (isPending) {
    return <p className="empty-state">正在校验管理员权限…</p>
  }

  if (isError) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <section className="card" data-testid="admin-denied">
        <div className="section-head">
          <h3>{forbidden ? '当前账号没有管理员权限' : '暂时无法校验管理员权限'}</h3>
        </div>
        <p className="muted">
          {forbidden
            ? '运营后台只对 ADMIN 账号开放。请退出后用管理员账号登录。'
            : '探测后台接口失败，请稍后重试或联系管理员。'}
        </p>
        <Link className="btn btn--inline" to="/">
          返回首页
        </Link>
      </section>
    )
  }

  return <>{children}</>
}
