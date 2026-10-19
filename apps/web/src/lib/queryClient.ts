import { QueryClient } from '@tanstack/react-query'

/** 全局查询客户端：失败重试 1 次、窗口聚焦不自动重取（作业演示场景更可控）。 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
})
