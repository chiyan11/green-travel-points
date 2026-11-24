import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
import { LinkProvider } from '@astryxdesign/core/Link'
import { Theme } from '@astryxdesign/core/theme'

// Astryx 样式表必须按 layer 顺序引入：字体 → reset → 组件基座 → 主题令牌。
import '@astryxdesign/theme-neutral/fonts.css'
import '@astryxdesign/core/reset.css'
import '@astryxdesign/core/astryx.css'
import '@/themes/green-travel/greenTravelTheme.css'
// 站点自有样式放在最后：它是无层（unlayered）CSS，天然覆盖 Astryx 的 @layer 规则。
import '@/styles.css'

import { greenTravelTheme } from '@/themes/green-travel/green-travel'
import { AstryxLink } from '@/components/AstryxLink'
import { queryClient } from '@/lib/queryClient'
import { router } from '@/routes'

const container = document.getElementById('root')
if (container === null) {
  throw new Error('找不到 #root 挂载点，请检查 index.html')
}

createRoot(container).render(
  <StrictMode>
    <Theme theme={greenTravelTheme} mode="light">
      <LinkProvider component={AstryxLink}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </LinkProvider>
    </Theme>
  </StrictMode>,
)
