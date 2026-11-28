import { describe, expect, it, vi, afterEach } from 'vitest'
import { newIdempotencyKey } from './idempotency'

/** 32 位小写十六进制——后端只当不透明字符串，前端固定长度便于排查。 */
const KEY_PATTERN = /^[0-9a-f]{32}$/

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('newIdempotencyKey', () => {
  it('有 randomUUID 时取 UUID 去掉连字符', () => {
    vi.stubGlobal('crypto', {
      randomUUID: () => 'ABCDEF01-2345-6789-ABCD-EF0123456789',
    } as unknown as Crypto)

    expect(newIdempotencyKey()).toBe('abcdef0123456789abcdef0123456789')
  })

  it('非安全上下文没有 randomUUID 时退回 getRandomValues', () => {
    // 局域网 http 演示就是这样：crypto 在，但 randomUUID 是 [SecureContext] 专属
    vi.stubGlobal('crypto', {
      getRandomValues: (bytes: Uint8Array) => {
        for (let i = 0; i < bytes.length; i += 1) bytes[i] = i
        return bytes
      },
    } as unknown as Crypto)

    expect(newIdempotencyKey()).toBe('000102030405060708090a0b0c0d0e0f')
  })

  it('两种随机源都没有时抛可读错误，而不是 TypeError', () => {
    vi.stubGlobal('crypto', undefined)

    expect(() => newIdempotencyKey()).toThrow('当前浏览器不支持生成兑换凭证')
  })

  it('生成的键满足 32 位十六进制格式', () => {
    expect(newIdempotencyKey()).toMatch(KEY_PATTERN)
  })
})
