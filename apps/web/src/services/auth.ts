/**
 * 认证服务（MOCK）。
 * TODO 联调替换：POST /api/v1/auth/login、POST /api/v1/auth/register，
 * Token 存入 Authorization: Bearer 头，见 docs/前端接口需求清单.md。
 */
import type { AuthUser } from '@/types/domain'

const SESSION_KEY = 'gtp_session'

export function loginMock(nickname: string): AuthUser {
  const user: AuthUser = {
    userId: 'u_demo',
    nickname: nickname.trim() || '演示用户',
    token: 'mock-token',
  }
  localStorage.setItem(SESSION_KEY, JSON.stringify(user))
  return user
}

export function getSession(): AuthUser | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY)
    if (!raw) return null
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}

export function logoutMock(): void {
  localStorage.removeItem(SESSION_KEY)
}
