/**
 * 登录会话存储（浏览器 localStorage）。
 *
 * 单独成文件是为了避免循环依赖：http.ts 需要读 token，auth.ts 需要写会话，
 * 两者都只依赖本模块。
 */
import type { AuthUser } from '@/types/domain'

const SESSION_KEY = 'gtp_session'

export function readSession(): AuthUser | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<AuthUser>
    if (!parsed || typeof parsed.token !== 'string' || typeof parsed.userId !== 'string') {
      return null
    }
    return { userId: parsed.userId, nickname: parsed.nickname ?? '', token: parsed.token }
  } catch {
    return null
  }
}

export function writeSession(user: AuthUser): void {
  localStorage.setItem(SESSION_KEY, JSON.stringify(user))
}

export function clearSession(): void {
  localStorage.removeItem(SESSION_KEY)
}

export function getToken(): string | null {
  return readSession()?.token ?? null
}
