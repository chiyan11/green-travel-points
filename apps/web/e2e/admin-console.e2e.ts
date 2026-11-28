import { expect, test, type Page } from '@playwright/test'

/**
 * M8 运营后台主链路（P5-3）：建商品 → 上架 → 消费者下单 → 后台看订单并确认接单。
 *
 * 账号默认来自本机开发库（scripts 之外的本地操作，不改 db/seed）：
 *   - 管理员 13800000002（user 1002，role=ADMIN）
 *   - 消费者 13800000001（user 1001，余额 1200 分）
 *
 * 跨机共享库（docs/dev/shared-db.md 里那台）的种子账号口令与本机开发库不同，
 * 所以这条用例默认只在「接口在本机」时跑；要对着共享库跑，就显式给出
 * E2E_ADMIN_PHONE / E2E_ADMIN_PASSWORD / E2E_CITIZEN_PHONE / E2E_CITIZEN_PASSWORD。
 */
const ADMIN = {
  phone: process.env.E2E_ADMIN_PHONE ?? '13800000002',
  password: process.env.E2E_ADMIN_PASSWORD ?? 'admin123456',
}
const CITIZEN = {
  phone: process.env.E2E_CITIZEN_PHONE ?? '13800000001',
  password: process.env.E2E_CITIZEN_PASSWORD ?? 'demo123456',
}

const accountsGiven = Boolean(process.env.E2E_ADMIN_PHONE || process.env.E2E_ADMIN_PASSWORD)

// 开跑前先探一次登录：本机开发库有这两个种子账号，跨机共享库上同 phone 的口令不同。
// 口令不对时直接跳过并说明原因，不要伪装成「页面卡在登录页」的产品缺陷。
let skipReason = ''

test.beforeAll(async ({ request }) => {
  if (accountsGiven) return
  for (const account of [ADMIN, CITIZEN]) {
    const res = await request.post('/api/v1/auth/login', { data: account })
    if (!res.ok()) {
      skipReason = `种子账号 ${account.phone} 在当前后端登录失败（HTTP ${res.status()}）：该用例默认依赖本机开发库；对着共享库跑请用 E2E_ADMIN_PHONE / E2E_ADMIN_PASSWORD / E2E_CITIZEN_PHONE / E2E_CITIZEN_PASSWORD 覆盖`
      return
    }
  }
})

async function login(page: Page, account: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('手机号').fill(account)
  await page.getByLabel('密码').fill(password)
  await page.locator('form').getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/$/)
  // 首页一版版改过文案，不再固定大标题；只断言登录完成后外壳（导航 + 退出）出现。
  await expect(page.getByRole('navigation', { name: '主导航' })).toBeVisible()
  await expect(page.getByRole('button', { name: '退出' })).toBeVisible()
}

test('建商品 → 上架 → 下单 → 后台看订单并确认接单', async ({ page, browser }) => {
  test.skip(Boolean(skipReason), skipReason)

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
  // 详情页现在是两步：先「确认兑换」打开确认弹窗，再点「确认兑换并生成核销码」才发请求。
  await citizenPage.getByRole('button', { name: '确认兑换' }).click()
  await citizenPage.getByRole('button', { name: '确认兑换并生成核销码' }).click()
  await expect(citizenPage).toHaveURL(/\/orders/)
  await expect(citizenPage.getByText(productName).first()).toBeVisible()

  await citizenPage.screenshot({
    path: '../../docs/p5-evidence/02-consumer-order.png',
    fullPage: true,
  })
  await citizenContext.close()

  // 4. 后台订单列表：看到该订单（待商户确认）→ 确认接单（PENDING → PAID）
  //    按钮现在先弹 window.confirm（目标状态 + 不可撤销）；Playwright 默认把弹窗 dismiss 掉，
  //    不接 dialog 的话请求不会发出，这里显式 accept 并把文案留下来断言。
  await page.goto('/admin/orders')
  const orderRow = page.getByRole('row', { name: new RegExp(productName) })
  await expect(orderRow).toBeVisible()
  await expect(orderRow.getByText('待商户确认')).toBeVisible()

  let confirmMessage = ''
  page.on('dialog', async (dialog) => {
    confirmMessage = dialog.message()
    await dialog.accept()
  })

  const [confirmResponse] = await Promise.all([
    page.waitForResponse(
      (res) => res.url().includes('/api/v1/admin/orders/') && res.url().endsWith('/confirm'),
    ),
    page.getByRole('row', { name: new RegExp(productName) })
      .getByRole('button', { name: '确认接单' })
      .click(),
  ])
  expect(confirmResponse.status(), `确认接单接口返回 ${confirmResponse.status()}`).toBe(200)
  expect(confirmMessage).toContain('不可撤销')
  expect(confirmMessage).toContain('已确认')
  await expect(
    page.getByRole('row', { name: new RegExp(productName) }).getByText(/已确认/),
  ).toBeVisible()

  await page.screenshot({
    path: '../../docs/p5-evidence/03-admin-orders-paid.png',
    fullPage: true,
  })
})
