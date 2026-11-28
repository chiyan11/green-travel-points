import type { ReactNode } from 'react'
import { IconSpark } from '@/components/icons'

interface QueryErrorBlockProps {
  /** 出错对象，用于取出后端文案；没有就显示兜底文案。 */
  error: unknown
  /** 重试回调，通常是 TanStack Query 的 `refetch`。 */
  onRetry: () => void
  /** 标题，默认「加载失败」。 */
  title?: string
}

export function errorMessage(error: unknown, fallback = '加载失败，请稍后重试'): string {
  if (error instanceof Error && error.message !== '') return error.message
  return fallback
}

/**
 * 查询失败的统一占位。
 *
 * 之前这些页面把「请求失败」和「没有数据」画成同一个空态（甚至显示 0 分），
 * 用户看不出是网络问题还是真的没内容，也不知道能重试。失败态必须单独一屏。
 */
export function QueryErrorBlock({ error, onRetry, title = '加载失败' }: QueryErrorBlockProps): ReactNode {
  return (
    <div className="empty" role="alert">
      <span className="empty__art">
        <IconSpark size={40} />
      </span>
      <p className="empty__title">{title}</p>
      <p className="empty__note">{errorMessage(error)}</p>
      <button type="button" className="btn btn--primary" onClick={onRetry}>
        重试
      </button>
    </div>
  )
}
