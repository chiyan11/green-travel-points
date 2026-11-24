import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

/**
 * 测试专用配置：复用 `vite.config.ts` 的插件与 `@` 别名，只覆盖测试期需要的东西。
 *
 * - `environment: 'jsdom'`：页面级测试要真的渲染 DOM（lib 层测试在 jsdom 下同样成立）。
 * - `setupFiles`：注册 jest-dom 断言、MSW 生命周期、相对路径 fetch 补丁。
 * - `css: false`：样式不参与断言，跳过 CSS 处理，测试更快。
 */
export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      // lib 层是 .test.ts、页面层是 .test.tsx，两种都要收。
      include: ['src/**/*.test.{ts,tsx}'],
      restoreMocks: true,
      css: false,
      // 页面里有 TanStack Query 的异步重试，测试里统一在 QueryClient 上关掉，不给全局超时兜底。
      testTimeout: 10000,
    },
  }),
)
