import { describe, expect, it } from 'vitest'
import { canTransition, ORDER_STATUS_META } from './orderStatus'
import { formatGrams, formatPoints } from './format'
import { capDaily, carbonSavedGrams, rawPointsFor, validateDistance } from './validators'

describe('validateDistance 距离区间 [0.1, 100] km', () => {
  it('空值与纯空白报错', () => {
    expect(validateDistance('')).toMatchObject({ ok: false })
    expect(validateDistance('   ')).toMatchObject({ ok: false })
  })

  it('非数字报错', () => {
    expect(validateDistance('abc')).toMatchObject({ ok: false })
  })

  it('低于下限 / 高于上限报错，端点 0.1 与 100 通过', () => {
    expect(validateDistance('0')).toMatchObject({ ok: false })
    expect(validateDistance('0.09')).toMatchObject({ ok: false })
    expect(validateDistance('100.1')).toMatchObject({ ok: false })
    const low = validateDistance('0.1')
    const high = validateDistance('100')
    expect(low).toMatchObject({ ok: true, value: 0.1 })
    expect(high).toMatchObject({ ok: true, value: 100 })
  })
})

describe('本地预览碳核算（与 domain 公式一致）', () => {
  it('地铁 3km：3 × (192−41) = 453g → floor(453/100) = 4 分', () => {
    expect(carbonSavedGrams(3, 'subway')).toBeCloseTo(453, 6)
    expect(rawPointsFor(453)).toBe(4)
  })

  it('步行 2km = 384g → 3 分；公交 10km = 1240g → 12 分', () => {
    expect(carbonSavedGrams(2, 'walk')).toBeCloseTo(384, 6)
    expect(rawPointsFor(384)).toBe(3)
    expect(carbonSavedGrams(10, 'bus')).toBeCloseTo(1240, 6)
    expect(rawPointsFor(1240)).toBe(12)
  })

  it('99.999g 不足 1 分；浮点累加 300g 不误判为 2 分', () => {
    expect(rawPointsFor(99.999)).toBe(0)
    // 0.1+0.2 浮点误差 ×1000 ≈ 300.00000000000006，容差后应得 3 分
    expect(rawPointsFor((0.1 + 0.2) * 1000)).toBe(3)
  })
})

describe('capDaily 单日 500 分封顶', () => {
  it('超出部分削平：600 分今日已得 0 → 实发 500', () => {
    expect(capDaily(600, 0)).toBe(500)
  })
  it('剩余额度封顶：已得 300，本次 300 → 实发 200', () => {
    expect(capDaily(300, 300)).toBe(200)
  })
  it('今日已达 500 → 0；cap=0 表示不限', () => {
    expect(capDaily(100, 500)).toBe(0)
    expect(capDaily(100, 0, 0)).toBe(100)
  })
})

describe('订单状态机合法跃迁', () => {
  it('PENDING → PAID 合法，PENDING → SHIPPED 非法', () => {
    expect(canTransition('PENDING', 'PAID')).toBe(true)
    expect(canTransition('PENDING', 'SHIPPED')).toBe(false)
  })
  it('SHIPPED → COMPLETED 合法，终态不可再流转', () => {
    expect(canTransition('SHIPPED', 'COMPLETED')).toBe(true)
    expect(canTransition('COMPLETED', 'SHIPPED')).toBe(false)
    expect(canTransition('CANCELLED', 'PAID')).toBe(false)
  })
  it('每个状态都有中文标签', () => {
    for (const status of Object.keys(ORDER_STATUS_META) as (keyof typeof ORDER_STATUS_META)[]) {
      expect(ORDER_STATUS_META[status].label.length).toBeGreaterThan(0)
    }
  })
})

describe('格式化', () => {
  it('克数自动换算 kg', () => {
    expect(formatGrams(453)).toBe('453.0 g')
    expect(formatGrams(1240)).toBe('1.24 kg')
  })
  it('积分千分位', () => {
    expect(formatPoints(1234)).toBe('1,234')
  })
})
