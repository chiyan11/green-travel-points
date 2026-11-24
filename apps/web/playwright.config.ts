import { defineConfig, devices } from '@playwright/test'

/**
 * E2E 配置：跑真实前后端，不做接口 mock。
 *
 * - 后端地址由 E2E_API_TARGET 决定（默认 8080，与开发口径一致），
 *   Vite dev server 通过 VITE_API_TARGET 把 /api 代理过去。
 * - 截图落在仓库根 docs/screenshots/，由用例自己调用 applyScreenshot 保存。
 */
const API_TARGET = process.env.E2E_API_TARGET ?? 'http://127.0.0.1:8080'
const WEB_PORT = Number(process.env.E2E_WEB_PORT ?? 5173)
const BASE_URL = `http://127.0.0.1:${WEB_PORT}`

export default defineConfig({
  testDir: './e2e',
  // 共享库跨 ZeroTier，写接口（POST /trips 一个事务里 8 条 SQL）单次要 10–20s，
  // 读接口 1–5s，所以超时给得比常规项目宽。
  timeout: 180_000,
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: Boolean(process.env.CI),
  reporter: [['list']],
  use: {
    baseURL: BASE_URL,
    viewport: { width: 1280, height: 900 },
    locale: 'zh-CN',
    timezoneId: 'Asia/Shanghai',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `npm run dev -- --port ${WEB_PORT} --strictPort`,
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: { VITE_API_TARGET: API_TARGET },
  },
})
