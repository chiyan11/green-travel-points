import { TRAVEL_MODES, type TravelMode } from '@/types/domain'

/** 出行方式的中文名（兜底显示原始 code）。 */
export function modeLabel(mode: TravelMode): string {
  return TRAVEL_MODES.find((m) => m.code === mode)?.label ?? mode
}

/** 出行方式的图标字符（兜底给一个中性点号）。 */
export function modeEmoji(mode: TravelMode): string {
  return TRAVEL_MODES.find((m) => m.code === mode)?.emoji ?? '·'
}
