/**
 * 原型内存数据仓库（MOCK）。
 * 联调后本文件整体删除：页面只依赖 src/services/* 的接口，不直接访问本模块。
 * 字段名已对齐 db/migrations/000003/000004/000005（见 types/domain.ts 注释）。
 */
import type { PointTxn, Product, RedeemOrder, TripRecord } from '@/types/domain'

let seq = 1000
export const nextId = (prefix: string): string => `${prefix}_${++seq}`

let balance = 1280

let trips: TripRecord[] = [
  {
    id: 'trip_1001',
    mode: 'subway',
    distanceKm: 3.2,
    startedAt: '2026-10-18T08:10:00+09:00',
    carbonSavedGrams: 483.2,
    rawPoints: 4,
    grantedPoints: 4,
    cappedByDaily: false,
    status: 'VALID',
  },
  {
    id: 'trip_1002',
    mode: 'bus',
    distanceKm: 8,
    startedAt: '2026-10-17T18:30:00+09:00',
    carbonSavedGrams: 992,
    rawPoints: 9,
    grantedPoints: 9,
    cappedByDaily: false,
    status: 'VALID',
  },
  {
    id: 'trip_1003',
    mode: 'walk',
    distanceKm: 5,
    startedAt: '2026-10-16T07:50:00+09:00',
    carbonSavedGrams: 960,
    rawPoints: 9,
    grantedPoints: 9,
    cappedByDaily: false,
    status: 'VALID',
  },
]

let txns: PointTxn[] = [
  {
    id: 'txn_1001',
    type: 'TRAVEL',
    amount: 4,
    balanceAfter: 1280,
    bizRefType: 'trip_record',
    bizRefId: 'trip_1001',
    occurredAt: '2026-10-18T08:10:05+09:00',
  },
  {
    id: 'txn_1002',
    type: 'REDEEM',
    amount: -300,
    balanceAfter: 980,
    bizRefType: 'redeem_order',
    bizRefId: 'order_1001',
    occurredAt: '2026-10-15T12:00:00+09:00',
  },
  {
    id: 'txn_1003',
    type: 'TRAVEL',
    amount: 9,
    balanceAfter: 989,
    bizRefType: 'trip_record',
    bizRefId: 'trip_1002',
    occurredAt: '2026-10-17T18:30:10+09:00',
  },
]

let products: Product[] = [
  { id: 'p_1', name: '环保帆布袋', description: '可折叠大容量帆布袋，减少一次性塑料袋使用。', category: '生活好物', coverUrl: '', status: 'ONLINE', pointsPrice: 300, stock: 20, limitPerUser: 0 },
  { id: 'p_2', name: '地铁月卡折扣券', description: '次月地铁乘车 9 折券，限本人使用。', category: '出行优惠', coverUrl: '', status: 'ONLINE', pointsPrice: 800, stock: 50, limitPerUser: 1 },
  { id: 'p_3', name: '桌面绿植盆栽', description: '绿萝一盆，净化桌面空气。', category: '生活好物', coverUrl: '', status: 'ONLINE', pointsPrice: 500, stock: 8, limitPerUser: 0 },
  { id: 'p_4', name: '保温水杯 500ml', description: '316 不锈钢保温杯，自带杯减少一次性杯。', category: '生活好物', coverUrl: '', status: 'ONLINE', pointsPrice: 650, stock: 12, limitPerUser: 0 },
  { id: 'p_5', name: '共享单车月卡', description: '30 天骑行不限次月卡。', category: '出行优惠', coverUrl: '', status: 'ONLINE', pointsPrice: 400, stock: 30, limitPerUser: 0 },
  { id: 'p_6', name: '再生纸笔记本', description: '80 页再生纸笔记本一本。', category: '学习文具', coverUrl: '', status: 'ONLINE', pointsPrice: 200, stock: 40, limitPerUser: 0 },
]

let orders: RedeemOrder[] = [
  { id: 'order_1001', orderNo: 'SO20261015001', productId: 'p_1', skuId: 'sku_1', skuName: '标准款', productName: '环保帆布袋', pointsSpent: 300, quantity: 1, receiver: '宋同学', status: 'COMPLETED', redeemCode: 'RC882100456', createdAt: '2026-10-15T12:00:00+09:00', updatedAt: '2026-10-16T10:00:00+09:00' },
  { id: 'order_1002', orderNo: 'SO20261019002', productId: 'p_5', skuId: 'sku_5', skuName: '30天月卡', productName: '共享单车月卡', pointsSpent: 400, quantity: 1, receiver: '宋同学', status: 'PENDING', redeemCode: '', createdAt: '2026-10-19T20:00:00+09:00', updatedAt: '2026-10-19T20:00:00+09:00' },
]

export const mockDb = {
  getBalance: () => balance,
  addPoints: (n: number) => {
    balance += n
  },
  getTrips: (): TripRecord[] => trips,
  addTrip: (t: TripRecord) => {
    trips = [t, ...trips]
  },
  getTxns: (): PointTxn[] => txns,
  addTxn: (t: PointTxn) => {
    txns = [t, ...txns]
  },
  getProducts: (): Product[] => products,
  getProduct: (id: string): Product | undefined => products.find((p) => p.id === id),
  decrementStock: (id: string) => {
    products = products.map((p) => (p.id === id && p.stock > 0 ? { ...p, stock: p.stock - 1 } : p))
  },
  restoreStock: (id: string) => {
    products = products.map((p) => (p.id === id ? { ...p, stock: p.stock + 1 } : p))
  },
  getOrders: (): RedeemOrder[] => orders,
  addOrder: (o: RedeemOrder) => {
    orders = [o, ...orders]
  },
  updateOrder: (id: string, patch: Partial<RedeemOrder>) => {
    orders = orders.map((o) => (o.id === id ? { ...o, ...patch, updatedAt: new Date().toISOString() } : o))
  },
  /** 今日已发放积分（原型：取当天 TRAVEL 流水之和）。 */
  earnedToday: (): number => {
    const today = new Date().toDateString()
    return txns
      .filter((t) => t.type === 'TRAVEL' && new Date(t.occurredAt).toDateString() === today)
      .reduce((sum, t) => sum + t.amount, 0)
  },
}
