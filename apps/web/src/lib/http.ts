/** 与 Go 后端 `internal/httpserver/response.go` 的 Envelope 保持一致的响应壳。 */
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

  const res = await fetch(path, { ...init, headers })
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
  post: <T>(path: string, body: unknown): Promise<T> =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
}

export interface PingResponse {
  env: string
  pong: string
}

/** Sprint 0 连通性自检：前端 → /api 代理 → Go 服务。 */
export const fetchPing = (): Promise<PingResponse> => api.get<PingResponse>('/api/v1/ping')
