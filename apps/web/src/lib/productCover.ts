/**
 * 商品封面渲染兜底：coverUrl（product.cover_url）为空时按分类给占位图标。
 * 联调后商品有真实图片 URL，直接 <img src={coverUrl}> 即可，本文件仅过渡期用。
 */
const FALLBACK_BY_CATEGORY: Record<string, string> = {
  生活好物: '🎁',
  出行优惠: '🚇',
  学习文具: '📓',
}

export function productCoverIcon(category: string): string {
  return FALLBACK_BY_CATEGORY[category] ?? '🎁'
}
