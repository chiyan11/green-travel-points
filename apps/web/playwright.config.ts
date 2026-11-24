import { defineConfig, devices } from '@playwright/test'

/**
 * E2E：消费者端主链路（`e2e/main-flow.spec.ts`）+ 运营后台主链路（`e2e/admin-console.e2e.ts`）。
 *
 * 两种跑法：
 *   1. 本机已有后端（比如连着共享库的实例）：
 *        E2E_API_TARGET=http://127.0.0.1:18080 npm run test:e2e
 *      此时只起 vite，代理指向该后端，不会再自启一个 `go run ./cmd/server`。
 *   2. 什么都不设：沿用后台 E2E 的方式，自动起 `apps/api` 的 `go run ./cmd/server`（默认 DSN）与 vite。
 *
 * 超时取自实测：共享库跨 ZeroTier，`POST /trips` 约 20s、`POST /orders` 约 12s，
 * 所以单测例上限放到 5 分钟、断言等待 2 分钟，避免把慢当失败。
 */
const API_TARGET = process.env.E2E_API_TARGET
const WEB_PORT = process.env.E2E_WEB_PORT ?? '5173'
const WEB_ORIGIN = `http://localhost:${WEB_PORT}`

const webServer = {
  command: `npm run dev -- --port ${WEB_PORT} --strictPort`,
  url: WEB_ORIGIN,
  reuseExistingServer: !process.env.CI,
  timeout: 120_000,
  env: { VITE_API_TARGET: API_TARGET ?? 'http://127.0.0.1:8080' },
}

const apiServer = {
  // 用 go run 而不是直接跑 .cache/bin/server.exe：Windows 的 cmd 解析相对 exe 路径不稳，
  // go run 与本地开发命令一致，Go 编译缓存热的情况下 2 秒内起好。
  command: 'go run ./cmd/server',
  cwd: '../../apps/api',
  url: 'http://127.0.0.1:8080/healthz',
  reuseExistingServer: !process.env.CI,
  timeout: 120_000,
}

export default defineConfig({
  testDir: './e2e',
  // 消费者端主链路用 *.spec.ts，后台主链路沿用 *.e2e.ts；两种都要被收进来。
  testMatch: ['**/*.e2e.ts', '**/*.spec.ts'],
  timeout: 300_000,
  expect: { timeout: 120_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? WEB_ORIGIN,
    actionTimeout: 120_000,
    locale: 'zh-CN',
    timezoneId: 'Asia/Shanghai',
    viewport: { width: 1280, height: 900 },
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: API_TARGET ? [webServer] : [apiServer, webServer],
})
