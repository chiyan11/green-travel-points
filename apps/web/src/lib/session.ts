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

/**
 * 写入会话。返回 false 表示浏览器拒绝写入（隐私模式、存储配额满、被策略禁用），
 * 调用方必须据此提示用户，而不是假装登录成功——否则下一次请求就没有 token 了。
 */
export function writeSession(user: AuthUser): boolean {
  try {
    localStorage.setItem(SESSION_KEY, JSON.stringify(user))
    return true
  } catch {
    return false
  }
}

export function clearSession(): void {
  try {
    localStorage.removeItem(SESSION_KEY)
  } catch {
    // 存储被禁用时本来就没有会话可清，不额外抛错打断登出流程
  }
}

export function getToken(): string | null {
  return readSession()?.token ?? null
}
