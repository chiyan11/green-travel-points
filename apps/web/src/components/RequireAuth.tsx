import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { getSession } from '@/services/auth'

/**
 * 路由守卫（MOCK）：无会话跳转 /login。
 * TODO 联调后改为校验 JWT 有效期，401 时由 http 层统一跳转。
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const session = getSession()
  if (session === null) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
