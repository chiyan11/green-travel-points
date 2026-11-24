import { defineConfig, devices } from '@playwright/test'

/**
 * P5 运营后台 E2E（M8）：一条「建商品 → 上架 → 下单 → 看订单」主链路。
 *
 * 前置：
 *   1. MySQL 里有 green_travel_points 库（本地默认 DSN 可直接连）；
 *   2. 已跑过 pwsh -File scripts/verify.ps1，产出 .cache/bin/server.exe；
 *   3. 浏览器：npx playwright install chromium。
 *
 * 两个服务都用 reuseExistingServer：本机已开着后端 / 前端时直接复用，不会抢 8080 / 5173。
 */
export default defineConfig({
  testDir: './e2e',
  // 用 *.e2e.ts 命名，避免 vitest 的默认 *.spec.ts / *.test.ts 命中这份用例。
  testMatch: '**/*.e2e.ts',
  timeout: 90_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    // vite 在 Windows 上只监听 [::1]:5173，用 127.0.0.1 会连不上，因此统一走 localhost。
    baseURL: 'http://localhost:5173',
    actionTimeout: 10_000,
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      // 用 go run 而不是直接跑 .cache/bin/server.exe：Windows 的 cmd 解析相对 exe 路径不稳，
      // go run 与本地开发命令一致，Go 编译缓存热的情况下 2 秒内起好。
      command: 'go run ./cmd/server',
      cwd: '../../apps/api',
      url: 'http://localhost:8080/healthz',
      reuseExistingServer: true,
      timeout: 60_000,
    },
    {
      command: 'npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: true,
      timeout: 60_000,
    },
  ],
})
