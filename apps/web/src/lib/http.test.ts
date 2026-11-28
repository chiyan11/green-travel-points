import { describe, expect, it, vi, afterEach } from 'vitest'
import { ApiError, REQUEST_TIMEOUT_MS, api, fetchPing, loginRedirectFrom } from './http'
import { clearSession, readSession, writeSession } from './session'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api.get', () => {
  it('code=OK 时返回 data', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({
      code: 'OK',
      message: 'ok',
      data: { pong: 'green-travel-points' },
    })))

    await expect(fetchPing()).resolves.toEqual({ pong: 'green-travel-points' })
  })

  it('业务错误码转换为 ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({
      code: 'INVALID_ARGUMENT',
      message: '距离必须大于 0',
    }, 400)))

    const err = await api.get('/api/v1/trips').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).code).toBe('INVALID_ARGUMENT')
    expect((err as ApiError).status).toBe(400)
    expect((err as ApiError).message).toBe('距离必须大于 0')
  })

  it('404 且响应为统一壳时保留 NOT_FOUND', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({
      code: 'NOT_FOUND',
      message: '接口不存在',
    }, 404)))

    await expect(api.get('/api/v1/nope')).rejects.toMatchObject({ code: 'NOT_FOUND', status: 404 })
  })

  it('响应体不是 JSON 时抛出 HTTP_ERROR', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502</html>', { status: 502 })))

    await expect(api.get('/api/v1/ping')).rejects.toMatchObject({ code: 'HTTP_ERROR', status: 502 })
  })

  it('POST 自动补 Content-Type 并序列化 body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ code: 'OK', message: 'ok', data: null }))
    vi.stubGlobal('fetch', fetchMock)

    await api.post('/api/v1/trips', { mode: 'subway', distanceKm: 3 })
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(init.method).toBe('POST')
    expect(init.body).toBe('{"mode":"subway","distanceKm":3}')
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json')
  })

  it('网络不可达时抛出中文 NETWORK_ERROR，而不是 Failed to fetch', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const err = (await api.get('/api/v1/points/summary').catch((e: unknown) => e)) as ApiError
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('NETWORK_ERROR')
    expect(err.message).toBe('网络异常，请检查连接后重试')
  })

  it('超过 65s 未返回时中止请求并给出超时文案', async () => {
    vi.useFakeTimers()
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_url: string, init?: RequestInit) =>
          new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () =>
              reject(new DOMException('aborted', 'AbortError')),
            )
          }),
      ),
    )

    const pending = api.get('/api/v1/points/summary').catch((e: unknown) => e)
    await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS + 1)
    const err = (await pending) as ApiError
    expect(err.code).toBe('TIMEOUT')
    expect(err.message).toBe('请求超时，请稍后重试')
    vi.useRealTimers()
  })

  it('401 会清掉本地会话，并给出带回跳地址的登录链接', async () => {
    writeSession({ userId: 'u_1', nickname: '测试', token: 'token-1' })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({
      code: 'UNAUTHORIZED',
      message: '需要登录',
    }, 401)))

    await expect(api.get('/api/v1/points/summary')).rejects.toMatchObject({ status: 401 })
    expect(readSession()).toBeNull()
    expect(loginRedirectFrom('/points?tab=1')).toBe('/login?next=%2Fpoints%3Ftab%3D1')
    clearSession()
  })
})
