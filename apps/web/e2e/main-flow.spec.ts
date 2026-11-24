import { test, expect, type Page } from '@playwright/test'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

/**
 * P4-2 主链路 E2E：真实后端 + 真实数据库，不做接口 mock。
 *
 * 覆盖 注册 → 上报行程 → 积分到账 → 商城兑换 → 订单取消返还，
 * 每一步都落一张截图到 docs/screenshots/，作为验收证据。
 */
const SHOT_DIR = fileURLToPath(new URL('../../../docs/screenshots', import.meta.url))

async function shot(page: Page, name: string): Promise<void> {
  await page.screenshot({ path: path.join(SHOT_DIR, name), fullPage: true })
}

/** 每次跑用新手机号，避免共享库里的历史数据影响断言。 */
function freshPhone(): string {
  return `139${Date.now().toString().slice(-8)}`
}

test('主链路：注册 → 上报行程 → 积分到账 → 商城兑换 → 订单取消', async ({ page }) => {
  const phone = freshPhone()

  // 1. 登录页
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '登录绿行积分' })).toBeVisible()
  await shot(page, '01-login.png')

  // 2. 注册并登录
  await page.getByRole('button', { name: '注册', exact: true }).click()
  await page.getByLabel('手机号').fill(phone)
  await page.getByLabel('昵称（可选）').fill('E2E 验收')
  await page.getByLabel('密码').fill('123456')
  await shot(page, '02-register.png')
  await page.getByRole('button', { name: '注册并登录' }).click()

  // 3. 首页
  await expect(page.getByRole('heading', { name: /把每一次低碳出行/ })).toBeVisible()
  await expect(page.getByRole('navigation', { name: '主导航' })).toBeVisible()
  // 首页右上的碳减排与积分卡依赖 summary 查询，等余额真的取回来再截图，否则会拍到占位符「—」
  await expect(page.locator('.balance-chip')).toHaveText(/可用\s*\d/, { timeout: 120_000 })
  await shot(page, '03-home.png')

  // 4. 上报两笔地铁 50km 行程（每笔 75 分，共 150 分，够换 100 分商品）
  // 写接口在共享库上要 12–20s，偶尔更久，所以等响应而不是只等 UI 变化。
  for (let i = 0; i < 2; i += 1) {
    await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '行程上报' }).click()
    await expect(page.getByRole('heading', { name: '行程上报' })).toBeVisible()
    await page.getByLabel(/距离（km/).fill('50')
    const [tripResponse] = await Promise.all([
      page.waitForResponse(
        (r) => new URL(r.url()).pathname.endsWith('/api/v1/trips') && r.request().method() === 'POST',
        { timeout: 180_000 },
      ),
      page.getByRole('button', { name: '上报行程' }).click(),
    ])
    expect(tripResponse.status(), `上报接口返回 ${tripResponse.status()}`).toBe(200)
    await expect(page.getByText('本次结果（后端核算）')).toBeVisible()
    await expect(page.getByText(/共 \d+ 条/)).toBeVisible()
    if (i === 0) await shot(page, '04-trips.png')
  }
  await expect(page.getByText('共 2 条')).toBeVisible()

  // 5. 积分到账
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '我的积分' }).click()
  await expect(page.getByRole('heading', { name: '我的积分' })).toBeVisible()
  const balanceCard = page.locator('.stat-card', { hasText: '当前余额' })
  await expect(balanceCard).toContainText('150 分', { timeout: 120_000 })
  // 流水是异步查询，等两笔流水都渲染出来再截图，否则会拍到加载态或空态
  await expect(page.locator('.list__row')).toHaveCount(2, { timeout: 120_000 })
  await shot(page, '05-points.png')

  // 6. 商城列表
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '积分商城' }).click()
  await expect(page.getByRole('heading', { name: '积分商城' })).toBeVisible()
  // 等商品卡片真的渲染出来再截图，否则会拍到「共 0 件」的空态
  await expect(page.locator('.product-card').first()).toBeVisible({ timeout: 120_000 })
  await expect(page.getByText(/共 [1-9]\d* 件在售商品/)).toBeVisible()
  await shot(page, '06-mall.png')

  // 7. 商品详情：挑一个 100 分、买得起的商品
  const card = page.locator('.product-card', { hasText: '环保帆布袋' }).first()
  await card.getByRole('link', { name: '查看详情' }).click()
  await expect(page.getByRole('button', { name: '确认兑换' })).toBeEnabled()
  await shot(page, '07-product-detail.png')
  // 兑换前先弹二次确认（弹窗里重申核销码规则），点确认才真的下单
  await page.getByRole('button', { name: '确认兑换', exact: true }).click()
  const confirmDialog = page.getByRole('dialog')
  await expect(confirmDialog).toBeVisible()
  const [redeemResponse] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/api/v1/orders') && r.request().method() === 'POST',
      { timeout: 120_000 },
    ),
    confirmDialog.getByRole('button', { name: '确认兑换并生成核销码' }).click(),
  ])
  expect(redeemResponse.status(), `兑换接口返回 ${redeemResponse.status()}`).toBe(201)

  // 8. 兑换成功落到兑换记录，新订单是「待支付」
  await expect(page).toHaveURL(/\/orders\?created=/)
  await expect(page.getByRole('heading', { name: '兑换记录' })).toBeVisible()
  await expect(page.getByText('共 1 条')).toBeVisible()
  await expect(page.getByRole('button', { name: '取消并返还' })).toBeVisible()
  // 顶栏余额走的是同一个 summary 查询，等它刷成 50 分再截图，否则会拍到兑换前的 150
  await expect(page.locator('.balance-chip')).toHaveText(/可用\s*50\s*分/, { timeout: 120_000 })
  await shot(page, '08-orders-created.png')

  // 9. 取消订单并返还积分（等取消接口真的回来，再断言状态，否则会和退款竞态）
  const [cancelResponse] = await Promise.all([
    page.waitForResponse(
      (r) => /\/api\/v1\/orders\/[^/]+\/cancel$/.test(new URL(r.url()).pathname) && r.request().method() === 'POST',
      { timeout: 120_000 },
    ),
    page.getByRole('button', { name: '取消并返还' }).click(),
  ])
  expect(cancelResponse.ok(), `取消接口返回 ${cancelResponse.status()}`).toBeTruthy()
  await expect(page.locator('.order-card').first()).toContainText('已取消')
  await expect(page.locator('.balance-chip')).toHaveText(/可用\s*150\s*分/, { timeout: 120_000 })
  await shot(page, '09-orders-cancelled.png')

  // 10. 积分回到 150
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '我的积分' }).click()
  await expect(page.locator('.stat-card', { hasText: '当前余额' })).toContainText('150 分', { timeout: 120_000 })
  await shot(page, '10-points-after-cancel.png')
})

test('登录失败：错误密码给出后端提示，不跳转首页', async ({ page }) => {
  // 用主链路里刚注册的账号不现实（跨用例无共享状态），这里直接打一个必然失败的组合。
  await page.goto('/')
  await page.getByLabel('手机号').fill('13900009999')
  await page.getByLabel('密码').fill('wrong-password')
  await page.locator('form').getByRole('button', { name: '登录' }).click()
  await expect(page.locator('.field-error')).toBeVisible()
  await expect(page.getByRole('heading', { name: '登录绿行积分' })).toBeVisible()
  await shot(page, '11-login-error.png')
})
