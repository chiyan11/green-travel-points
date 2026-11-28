import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearSession, getToken, readSession, writeSession } from './session'

const user = { userId: 'u_1005', nickname: '宋昊阳', token: 'token-1' }

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

describe('session', () => {
  it('写入后能读回，并作为 token 来源', () => {
    expect(writeSession(user)).toBe(true)
    expect(readSession()).toEqual(user)
    expect(getToken()).toBe('token-1')
  })

  it('浏览器拒绝写入时返回 false，而不是把异常抛给调用方', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })

    expect(writeSession(user)).toBe(false)
    expect(readSession()).toBeNull()
  })

  it('存的内容不合法时按未登录处理', () => {
    localStorage.setItem('gtp_session', '{not-json')
    expect(readSession()).toBeNull()

    localStorage.setItem('gtp_session', JSON.stringify({ userId: 'u_1' }))
    expect(readSession()).toBeNull()
  })

  it('存储被禁用时清除会话不抛错', () => {
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => {
      throw new Error('SecurityError')
    })

    expect(() => clearSession()).not.toThrow()
  })
})
