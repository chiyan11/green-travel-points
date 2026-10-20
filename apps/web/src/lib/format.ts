/** 展示层格式化：积分、克数、时间。纯函数，便于单测。 */

export function formatPoints(n: number): string {
  return n.toLocaleString('zh-CN')
}

/** 克数自动换算 kg，保留两位小数。 */
export function formatGrams(grams: number): string {
  if (grams >= 1000) return `${(grams / 1000).toFixed(2)} kg`
  return `${grams.toFixed(1)} g`
}

/** ISO 时间 → 本地 YYYY-MM-DD HH:mm。 */
export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 距离展示：去掉无意义的尾零。 */
export function formatDistance(km: number): string {
  return `${km} km`
}
