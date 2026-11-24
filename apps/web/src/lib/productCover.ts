/**
 * 商品封面渲染兜底：`coverUrl` 为空时按品类给一个线性图标。
 *
 * 后端品类是自由文本（现在库里是「文创 / 餐饮 / 装备」），所以按关键词归类，
 * 认不出来的一律回落成礼物盒，不要因为多了一个品类就渲染成空白。
 */
export type ProductCoverKind = 'gift' | 'ticket' | 'package'

const RULES: ReadonlyArray<readonly [RegExp, ProductCoverKind]> = [
  [/券|票|卡|咖啡|饮|餐|兑换/, 'ticket'],
  [/头盔|装备|骑行|文具|书|本|器材/, 'package'],
  [/袋|包|文创|生活|好物|日用品/, 'gift'],
]

export function productCoverKind(category: string | undefined): ProductCoverKind {
  if (!category) return 'gift'
  for (const [pattern, kind] of RULES) {
    if (pattern.test(category)) return kind
  }
  return 'gift'
}
