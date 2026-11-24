import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
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
})
