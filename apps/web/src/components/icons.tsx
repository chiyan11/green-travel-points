/**
 * 单色线性图标（24×24，stroke 1.75，currentColor）。
 *
 * 只用于界面视觉：入口卡、空态插画、登录输入框前缀、商品占位图。
 * 这里手写而不是引入图标库，是为了不因为「装饰」把依赖面扩大。
 */
import type { ReactNode, SVGProps } from 'react'

type IconProps = SVGProps<SVGSVGElement> & { size?: number }

function icon(path: ReactNode, props: IconProps) {
  const { size = 24, ...rest } = props
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...rest}
    >
      {path}
    </svg>
  )
}

export const IconLeaf = (p: IconProps) =>
  icon(
    <>
      <path d="M4 20c8 0 15-5 15-15-8 0-15 5-15 15Z" />
      <path d="M4 20c2-6 6-10 12-12" />
    </>,
    p,
  )

export const IconRoute = (p: IconProps) =>
  icon(
    <>
      <circle cx="6" cy="18" r="2.2" />
      <circle cx="18" cy="6" r="2.2" />
      <path d="M8 17.5c4 0 4-4 8-4s2-4 2-6" />
    </>,
    p,
  )

export const IconAward = (p: IconProps) =>
  icon(
    <>
      <circle cx="12" cy="9" r="5" />
      <path d="M9 13.5 8 21l4-2 4 2-1-7.5" />
    </>,
    p,
  )

export const IconGift = (p: IconProps) =>
  icon(
    <>
      <rect x="4" y="9" width="16" height="11" rx="2" />
      <path d="M4 13h16M12 9v11" />
      <path d="M12 9c0-2.5-1.5-4-3.5-3.5C7 6 7.5 9 12 9Zm0 0c0-2.5 1.5-4 3.5-3.5C17 6 16.5 9 12 9Z" />
    </>,
    p,
  )

export const IconReceipt = (p: IconProps) =>
  icon(
    <>
      <path d="M6 3h12v18l-3-2-3 2-3-2-3 2V3Z" />
      <path d="M9 8h6M9 12h6" />
    </>,
    p,
  )

export const IconUser = (p: IconProps) =>
  icon(
    <>
      <circle cx="12" cy="8.5" r="3.5" />
      <path d="M5 20c1.2-3.6 3.8-5.2 7-5.2s5.8 1.6 7 5.2" />
    </>,
    p,
  )

export const IconArrowRight = (p: IconProps) =>
  icon(
    <>
      <path d="M5 12h13" />
      <path d="m13 6 6 6-6 6" />
    </>,
    p,
  )

export const IconCheck = (p: IconProps) =>
  icon(
    <>
      <path d="m5 12.5 4.5 4.5L19 7" />
    </>,
    p,
  )

export const IconPhone = (p: IconProps) =>
  icon(
    <>
      <rect x="7" y="2.5" width="10" height="19" rx="2.5" />
      <path d="M11 5.5h2" />
    </>,
    p,
  )

export const IconLock = (p: IconProps) =>
  icon(
    <>
      <rect x="4.5" y="10" width="15" height="10.5" rx="2.5" />
      <path d="M8.5 10V7.5a3.5 3.5 0 0 1 7 0V10" />
    </>,
    p,
  )

export const IconSpark = (p: IconProps) =>
  icon(
    <>
      <path d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1" />
    </>,
    p,
  )

/** 空态插画：一条没有脚印的路径。 */
export const IconEmptyRoute = (p: IconProps) =>
  icon(
    <>
      <path d="M3 17c3.5 0 4-8 8-8s4.5 8 10 8" strokeDasharray="3 4" />
      <circle cx="8" cy="20" r="1.2" fill="currentColor" stroke="none" />
      <circle cx="16" cy="20" r="1.2" fill="currentColor" stroke="none" />
    </>,
    p,
  )

/** 商品占位图：拎着的购物袋。 */
export const IconPackage = (p: IconProps) =>
  icon(
    <>
      <path d="M5.5 8h13l-1.2 12.2a1.5 1.5 0 0 1-1.5 1.3H8.2a1.5 1.5 0 0 1-1.5-1.3L5.5 8Z" />
      <path d="M9 8V6.2a3 3 0 0 1 6 0V8" />
    </>,
    p,
  )

/** 商品占位图：券。 */
export const IconTicket = (p: IconProps) =>
  icon(
    <>
      <path d="M3.5 9.5a2.5 2.5 0 0 1 0 5V17a1.5 1.5 0 0 0 1.5 1.5h14A1.5 1.5 0 0 0 20.5 17v-2.5a2.5 2.5 0 0 1 0-5V7A1.5 1.5 0 0 0 19 5.5H5A1.5 1.5 0 0 0 3.5 7v2.5Z" />
      <path d="M12 8v8" strokeDasharray="2.5 3" />
    </>,
    p,
  )

/** 商品占位图：头盔 / 装备类。 */
export const IconHelmet = (p: IconProps) =>
  icon(
    <>
      <path d="M4 15a8 8 0 0 1 16 0v2a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 17v-2Z" />
      <path d="M4 15h16" />
    </>,
    p,
  )

export const IconTrendUp = (p: IconProps) =>
  icon(
    <>
      <path d="m4 16 6-6 4 4 6-7" />
      <path d="M15 7h5v5" />
    </>,
    p,
  )

export const IconTrendDown = (p: IconProps) =>
  icon(
    <>
      <path d="m4 8 6 6 4-4 6 7" />
      <path d="M15 17h5v-5" />
    </>,
    p,
  )
