/**
 * 认证服务。
 * POST /api/v1/auth/register、POST /api/v1/auth/login、GET /api/v1/me。
 *
 * 传输类型来自 `src/types/api.ts`（由 docs/api/openapi.yaml 生成）。
 * 后端约定（apps/api/internal/httpserver/auth.go）：
 * - 注册/登录请求体只允许 phone / account / nickname / password 四个键（DisallowUnknownFields）
 * - 注册用手机号登记（phone 或 account 传一个即可），昵称可省略（默认同手机号）
 * - 登录按手机号查库
 * - 响应统一壳 {code, message, data}
 */
import { api } from '@/lib/http'
import { clearSession, readSession, writeSession } from '@/lib/session'
import type { AuthSession, LoginResult, MeResult, RegisterInput } from '@/types/api'
import type { AuthUser } from '@/types/domain'

/** 登录：账号即手机号。成功后把 token 与用户信息写入本地会话。 */
export async function login(account: string, password: string): Promise<AuthUser> {
  const res = await api.post<LoginResult>('/api/v1/auth/login', {
    account: account.trim(),
    password,
  })
  const user: AuthUser = {
    userId: res.user.userId,
    nickname: res.user.nickname,
    token: res.token,
  }
  writeSession(user)
  return user
}

/** 注册：手机号 + 密码（≥6 位）+ 可选昵称；成功后直接带 token 登录态。 */
export async function register(input: {
  phone: string
  password: string
  nickname?: string
}): Promise<AuthUser> {
  const body: RegisterInput = {
    phone: input.phone.trim(),
    password: input.password,
  }
  const nickname = input.nickname?.trim()
  if (nickname) body.nickname = nickname

  const res = await api.post<AuthSession>('/api/v1/auth/register', body)
  const user: AuthUser = {
    userId: res.userId,
    nickname: res.nickname,
    token: res.token,
  }
  writeSession(user)
  return user
}

export type { MeResult as MeResponse }

/** 校验当前 token 是否有效（401 时 http 层会自动清会话并跳登录）。 */
export async function fetchMe(): Promise<MeResult> {
  return api.get<MeResult>('/api/v1/me')
}

export function getSession(): AuthUser | null {
  return readSession()
}

export function logout(): void {
  clearSession()
}
