import { expect, test, type Page } from '@playwright/test'

/**
 * M8 运营后台主链路（P5-3）：建商品 → 上架 → 消费者下单 → 后台看订单并确认收款。
 *
 * 账号来自本机开发库（scripts 之外的本地操作，不改 db/seed）：
 *   - 管理员 13800000002（user 1002，role=ADMIN）
 *   - 消费者 13800000001（user 1001，余额 1200 分）
 */
const ADMIN = { phone: '13800000002', password: 'admin123456' }
const CITIZEN = { phone: '13800000001', password: 'demo123456' }

async function login(page: Page, account: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('手机号').fill(account)
  await page.getByLabel('密码').fill(password)
  await page.locator('form').getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: '绿行积分' })).toBeVisible()
}

test('建商品 → 上架 → 下单 → 后台看订单并确认收款', async ({ page, browser }) => {
  const productName = `E2E环保水杯-${Date.now()}`

  // 1. 管理员登录，新建商品（默认 DRAFT）
  await login(page, ADMIN.phone, ADMIN.password)
  await page.goto('/admin/products/new')
  await page.getByLabel('名称').fill(productName)
  await page.getByLabel('分类').fill('E2E')
  await page.getByLabel('描述').fill('端到端用例创建的测试商品')
  await page.getByLabel('积分价（标准款 SKU）').fill('100')
  await page.getByLabel('库存').fill('3')
  await page.getByRole('button', { name: '创建商品' }).click()

  await expect(page).toHaveURL(/\/admin\/products\?created=/)
  await expect(page.getByText(`已创建：${productName}`)).toBeVisible()

  const productRow = page.getByRole('row', { name: new RegExp(productName) })
  await expect(productRow).toBeVisible()
  await expect(productRow.getByText('草稿')).toBeVisible()

  // 2. 上架（DRAFT → ONLINE）
  await productRow.getByRole('button', { name: '上架' }).click()
  await expect(productRow.getByText('已上架')).toBeVisible()

  // 3. 改库存（展开行 → 输入 5 → 保存，总库存 3 → 5）
  await productRow.getByRole('button', { name: '改库存' }).click()
  const stockInput = page.locator('.sku-panel input').first()
  await expect(stockInput).toHaveValue('3')
  await stockInput.fill('5')
  await page.getByRole('button', { name: '保存' }).click()
  await expect(productRow.locator('td').nth(5)).toHaveText('5')

  await page.screenshot({
    path: '../../docs/p5-evidence/01-admin-products.png',
    fullPage: true,
  })

  // 3. 消费者登录：商城看到新商品 → 下单
  const citizenContext = await browser.newContext()
  const citizenPage = await citizenContext.newPage()
  await login(citizenPage, CITIZEN.phone, CITIZEN.password)
  await citizenPage.goto('/mall')

  const card = citizenPage.locator('.product-card', { hasText: productName })
  await expect(card).toBeVisible()
  await card.getByRole('link', { name: '查看详情' }).click()

  await expect(citizenPage.getByRole('heading', { name: productName })).toBeVisible()
  await citizenPage.getByRole('button', { name: '确认兑换' }).click()
  await expect(citizenPage).toHaveURL(/\/orders/)
  await expect(citizenPage.getByText(productName).first()).toBeVisible()

  await citizenPage.screenshot({
    path: '../../docs/p5-evidence/02-consumer-order.png',
    fullPage: true,
  })
  await citizenContext.close()

  // 4. 后台订单列表：看到该订单（待支付）→ 确认收款（PENDING → PAID）
  await page.goto('/admin/orders')
  const orderRow = page.getByRole('row', { name: new RegExp(productName) })
  await expect(orderRow).toBeVisible()
  await expect(orderRow.getByText('待支付')).toBeVisible()

  await page.getByRole('row', { name: new RegExp(productName) })
    .getByRole('button', { name: '确认收款' })
    .click()
  await expect(
    page.getByRole('row', { name: new RegExp(productName) }).getByText('已支付'),
  ).toBeVisible()

  await page.screenshot({
    path: '../../docs/p5-evidence/03-admin-orders-paid.png',
    fullPage: true,
  })
})
