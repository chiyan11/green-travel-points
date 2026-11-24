import { setupServer } from 'msw/node'

/**
 * 全局 MSW server：默认没有任何 handler，所有未注册的请求都会被当成错误。
 * 各测试文件用 `server.use(...)` 声明自己关心的端点，这样用例之间互不影响，
 * 也不会出现「忘了 stub 却静默通过」的情况。
 */
export const server = setupServer()
