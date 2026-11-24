import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './server'

/**
 * 测试环境补齐（页面级测试依赖三件事）：
 *
 * 1. `fetch` 只接受绝对 URL：生产代码用 `/api/v1/...` 这类相对路径（浏览器里由同源解析），
 *    Node 的 fetch 会直接抛 `Failed to parse URL`，这里按 jsdom 的 origin 补齐。
 * 2. MSW 拦截：默认对**未注册**的请求报错，避免测试悄悄打到真后端或真的网络。
 * 3. 每个用例之间清 DOM、清 localStorage（会话 token）、重置 handler 覆盖。
 */
const nativeFetch = globalThis.fetch

globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
  if (typeof input === 'string' && input.startsWith('/')) {
    return nativeFetch(new URL(input, window.location.origin).toString(), init)
  }
  return nativeFetch(input as RequestInfo, init)
}) as typeof fetch

beforeAll(() => {
  // MSW 3 把 onUnhandledRequest 改名为 onUnhandledFrame：未注册的请求直接报错，
  // 避免测试悄悄打到真后端。加接口时记得补 handler。
  server.listen({ onUnhandledFrame: 'error' })
})

afterEach(() => {
  cleanup()
  server.resetHandlers()
  localStorage.clear()
})

afterAll(() => {
  server.close()
})
