import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
// Astryx 基础样式（全部在 @layer reset / astryx-base / astryx-theme 内）
import '@astryxdesign/core/reset.css'
import '@astryxdesign/core/astryx.css'
// 主题：astryx-themes.ts 自带 theme.css 与 fonts.css 的副作用导入
import { Theme } from '@astryxdesign/core/theme'
import { themes, defaultThemeSlug } from '@/astryx-themes'
// 品牌色覆盖与页面级样式（不分层，优先级高于 Astryx 的 @layer）
import '@/astryx-brand.css'
import '@/styles.css'
import { queryClient } from '@/lib/queryClient'
import { router } from '@/routes'

const container = document.getElementById('root')
if (container === null) {
  throw new Error('找不到 #root 挂载点，请检查 index.html')
}

createRoot(container).render(
  <StrictMode>
    {/* mode 固定 light：全站是浅色稿，默认 system 会在深色系统上渲染成暗色 */}
    <Theme theme={themes[defaultThemeSlug]} mode="light">
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </Theme>
  </StrictMode>,
)
