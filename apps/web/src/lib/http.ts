/** 与 Go 后端 `internal/httpserver/response.go` 的 Envelope 保持一致的响应壳。 */
import type { PingResult } from '@/types/api'
import { clearSession, getToken } from './session'

export interface Envelope<T> {
  code: string
  message: string
  data?: T
}

export const CodeOK = 'OK'

/** 后端业务错误：code 为统一错误码，status 为 HTTP 状态码。 */
export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

/**
 * 单请求上限。共享库在异地（ZeroTier），写接口实测 10–40s，
 * 这里留到 65s：比后端 `HTTP_REQUEST_TIMEOUT`(90s) 小，但要能兜住「请求永远不返回」。
 */
export const REQUEST_TIMEOUT_MS = 65_000

/** 401 后回登录页的地址：带上来源页，登录成功后能跳回去。 */
export function loginRedirectFrom(from: string): string {
  return `/login?next=${encodeURIComponent(from)}`
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  // 只走 Bearer：后端默认关闭开发身份头（AUTH_ALLOW_DEV_HEADER），
  // 未登录时匿名请求由后端返回 401，再由下面的统一处理跳 /login。
  const token = getToken()
  if (token !== null && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)

  let res: Response
  try {
    res = await fetch(path, { ...init, headers, signal: controller.signal })
  } catch {
    // 浏览器对 fetch 失败的原始错误是 TypeError('Failed to fetch')，用户看不懂；
    // 超时被我们自己的 controller 中止，要给出可区分的文案。
    if (controller.signal.aborted) {
      throw new ApiError('请求超时，请稍后重试', 'TIMEOUT', 0)
    }
    throw new ApiError('网络异常，请检查连接后重试', 'NETWORK_ERROR', 0)
  } finally {
    clearTimeout(timer)
  }

  // 会话失效统一处理：清本地会话并回登录页，同时把来源页带上，
  // 登录成功后能回到用户原本想看的页面（避免 401 之后一律回首页）。
  if (res.status === 401 && !path.startsWith('/api/v1/auth/')) {
    clearSession()
    if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
      const from = `${window.location.pathname}${window.location.search}`
      window.location.assign(loginRedirectFrom(from))
    }
  }

  const text = await res.text()

  let payload: Envelope<T> | null = null
  if (text.length > 0) {
    try {
      payload = JSON.parse(text) as Envelope<T>
    } catch {
      payload = null
    }
  }

  if (!res.ok || payload === null || payload.code !== CodeOK) {
    throw new ApiError(
      payload?.message ?? `请求失败（HTTP ${res.status}）`,
      payload?.code ?? 'HTTP_ERROR',
      res.status,
    )
  }
  return payload.data as T
}

export const api = {
  get: <T>(path: string): Promise<T> => request<T>(path),
  post: <T>(path: string, body: unknown, headers?: Record<string, string>): Promise<T> =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body), headers }),
  // PATCH / PUT 供运营后台（商品上下架、SKU 库存）使用。
  patch: <T>(path: string, body: unknown): Promise<T> =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown): Promise<T> =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
}

export type PingResponse = PingResult

/** Sprint 0 连通性自检：前端 → /api 代理 → Go 服务。 */
export const fetchPing = (): Promise<PingResult> => api.get<PingResult>('/api/v1/ping')
