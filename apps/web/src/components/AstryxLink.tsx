import type { CSSProperties, MouseEventHandler, ReactNode } from 'react'
import { Link as RouterLink } from 'react-router-dom'

interface AstryxLinkProps {
  /** Astryx 组件传下来的目标地址；内部转成 react-router 的 to。 */
  href?: string
  to?: string
  children?: ReactNode
  className?: string
  style?: CSSProperties
  onClick?: MouseEventHandler<HTMLAnchorElement>
  target?: string
  rel?: string
  'aria-current'?: 'page' | 'step' | 'location' | 'date' | 'time' | 'true' | 'false' | boolean
  'aria-label'?: string
}

/**
 * 交给 Astryx `LinkProvider` 的链接实现：把 Astryx 组件里的 href 接到 react-router，
 * 站内跳转不再触发整页刷新。
 */
export function AstryxLink({ href, to, children, ...rest }: AstryxLinkProps) {
  return (
    <RouterLink to={href ?? to ?? '/'} {...rest}>
      {children}
    </RouterLink>
  )
}
