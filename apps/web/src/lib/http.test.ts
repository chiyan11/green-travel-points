import { describe, expect, it, vi, afterEach } from 'vitest'
import { ApiError, api, fetchPing } from './http'

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
      data: { env: 'dev', pong: 'green-travel-points' },
    })))

    await expect(fetchPing()).resolves.toEqual({ env: 'dev', pong: 'green-travel-points' })
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
})
