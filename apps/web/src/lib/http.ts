/** 与 Go 后端 `internal/httpserver/response.go` 的 Envelope 保持一致的响应壳。 */
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

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  // 已登录走 JWT（后端 httpserver/auth.go Require 优先校验 Bearer）；
  // 未登录时退回开发期约定的 X-User-Id，方便不登录也能浏览商城。
  const token = getToken()
  if (token !== null && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  } else if (token === null && !headers.has('X-User-Id')) {
    headers.set('X-User-Id', '1')
  }

  const res = await fetch(path, { ...init, headers })

  // 会话失效统一处理：清本地会话并回登录页（避免多页重复写判断）。
  if (res.status === 401 && !path.startsWith('/api/v1/auth/')) {
    clearSession()
    if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
      window.location.assign('/login')
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

export interface PingResponse {
  env: string
  pong: string
}

/** Sprint 0 连通性自检：前端 → /api 代理 → Go 服务。 */
export const fetchPing = (): Promise<PingResponse> => api.get<PingResponse>('/api/v1/ping')
