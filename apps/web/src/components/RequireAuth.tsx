import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { getSession } from '@/services/auth'

/**
 * 路由守卫：无本地会话则跳转 /login。
 * 会话过期由 http 层统一处理（401 时清会话并跳登录）。
 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const session = getSession()
  if (session === null) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
