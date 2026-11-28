import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { LoginPage } from '@/pages/LoginPage'
import { renderPage } from '@/test/render'
import { failJson, okJson } from '@/test/http'
import { server } from '@/test/server'
import { readSession } from '@/lib/session'

/**
 * 登录页要挂在 `/login`：它成功后 `navigate('/')`，只有被测页不在 `/` 时，
 * `/` 才会落到渲染壳的兜底路由，从而能断言「确实跳走了」。
 */
function renderLogin() {
  return renderPage(<LoginPage />, { route: '/login', path: '/login' })
}

/** 页面上有个 tab 也叫「登录」，用表单作用域定位真正的提交按钮。 */
function formOf(): HTMLFormElement {
  return screen.getByLabelText('手机号').closest('form') as HTMLFormElement
}

async function fillCredentials(account: string, password: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('手机号'), account)
  await user.type(screen.getByLabelText('密码'), password)
  return user
}

describe('LoginPage', () => {
  it('空表单提交时给前端校验提示，不发请求', async () => {
    const user = userEvent.setup()
    renderLogin()

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByText('请填写手机号和密码')).toBeInTheDocument()
  })

  it('密码少于 6 位时在本地拦下', async () => {
    renderLogin()
    const user = await fillCredentials('13900001111', '123')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByText('密码至少 6 位')).toBeInTheDocument()
  })

  it('后端返回 401 时展示后端文案，且不清空输入', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        failJson(401, 'UNAUTHORIZED', '手机号或密码错误'),
      ),
    )
    renderLogin()
    const user = await fillCredentials('13900001111', '000000')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByText('手机号或密码错误')).toBeInTheDocument()
    expect(screen.getByLabelText('手机号')).toHaveValue('13900001111')
    expect(readSession()).toBeNull()
  })

  it('登录成功后写入会话并跳转首页', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        okJson({ token: 'token-abc', user: { userId: 'u_1005', nickname: '宋昊阳' } }),
      ),
    )
    renderLogin()
    const user = await fillCredentials('13900001111', '123456')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
    expect(readSession()).toEqual({ userId: 'u_1005', nickname: '宋昊阳', token: 'token-abc' })
  })

  it('带 ?next= 登录后回到来源页，而不是一律回首页', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        okJson({ token: 'token-1', user: { userId: 'u_1', nickname: '测试' } }),
      ),
    )
    renderPage(<LoginPage />, { route: '/login?next=%2Fpoints', path: '/login' })
    const user = await fillCredentials('13800000001', '123456')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    const stub = await screen.findByTestId('route-stub')
    expect(stub).toHaveAttribute('data-path', '/points')
  })

  it('?next= 是站外地址时回首页，不作为开放重定向', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        okJson({ token: 'token-1', user: { userId: 'u_1', nickname: '测试' } }),
      ),
    )
    renderPage(<LoginPage />, { route: '/login?next=%2F%2Fevil.example.com', path: '/login' })
    const user = await fillCredentials('13800000001', '123456')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    const stub = await screen.findByTestId('route-stub')
    expect(stub).toHaveAttribute('data-path', '/')
  })

  it('浏览器存储不可用时提示无法保持登录，且不跳首页', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        okJson({ token: 'token-1', user: { userId: 'u_1', nickname: '测试' } }),
      ),
    )
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    renderLogin()
    const user = await fillCredentials('13800000001', '123456')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByText(/浏览器存储不可用/)).toBeInTheDocument()
    expect(screen.queryByTestId('route-stub')).not.toBeInTheDocument()
    expect(readSession()).toBeNull()
    vi.restoreAllMocks()
  })

  it('会话写入成功后才算登录完成', async () => {
    server.use(
      http.post('*/api/v1/auth/login', () =>
        okJson({ token: 'token-1', user: { userId: 'u_1', nickname: '测试' } }),
      ),
    )
    renderLogin()
    const user = await fillCredentials('13800000001', '123456')

    await user.click(within(formOf()).getByRole('button', { name: '登录' }))

    expect(await screen.findByTestId('route-stub')).toBeInTheDocument()
    expect(readSession()?.token).toBe('token-1')
  })
})
