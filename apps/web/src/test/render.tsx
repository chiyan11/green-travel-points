import type { ReactElement, ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

/**
 * 页面级测试的公共渲染壳：QueryClient + 路由。
 *
 * - `retry: false`：后端返回错误时立刻进入错误态，测试不需要等重试间隔。
 * - `gcTime: 0`：每个用例一个全新 QueryClient，避免缓存跨用例泄漏。
 * - 未匹配的路由落到 `route-stub`，用来断言「提交成功后跳走了」（例如登录成功回首页）。
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  })
}

export interface RenderPageOptions {
  /** 当前地址，默认 `/`。 */
  route?: string
  /** 被测页面挂载的路由模板，默认 `/`（带参数页面用 `/mall/:id`）。 */
  path?: string
}

export function renderPage(ui: ReactElement, options: RenderPageOptions = {}) {
  const { route = '/', path = '/' } = options
  const queryClient = createTestQueryClient()

  const result = render(ui, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[route]}>
          <Routes>
            <Route path={path} element={children as ReactElement} />
            <Route path="*" element={<div data-testid="route-stub">已离开当前页</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  })

  return { ...result, queryClient }
}
