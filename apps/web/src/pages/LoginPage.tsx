import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@astryxdesign/core/Button'
import { Card } from '@astryxdesign/core/Card'
import { TextInput } from '@astryxdesign/core/TextInput'
import { VStack } from '@astryxdesign/core/VStack'
import { login, register } from '@/services/auth'
import { ApiError } from '@/lib/http'

type Tab = 'login' | 'register'

/**
 * 登录 / 注册页。
 * 后端按手机号登记账号：注册可填昵称（省略则用手机号），登录用手机号 + 密码。
 */
export function LoginPage(): ReactNode {
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('login')
  const [account, setAccount] = useState('')
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    if (account.trim() === '' || password === '') {
      setError('请填写手机号和密码')
      return
    }
    if (password.length < 6) {
      setError('密码至少 6 位')
      return
    }
    setPending(true)
    try {
      if (tab === 'login') {
        await login(account, password)
      } else {
        await register({ phone: account, password, nickname })
      }
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '网络异常，请确认后端已启动')
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="auth-wrap">
      <section className="hero">
        <h2>登录绿行积分</h2>
        <p>上报绿色出行，积攒碳积分，兑换绿色好物。</p>
      </section>

      <Card padding={5} elevation="low">
        <VStack gap={3}>
          <div className="filter-bar">
            <button
              type="button"
              className={tab === 'login' ? 'chip chip--active' : 'chip'}
              onClick={() => {
                setTab('login')
                setError('')
              }}
            >
              登录
            </button>
            <button
              type="button"
              className={tab === 'register' ? 'chip chip--active' : 'chip'}
              onClick={() => {
                setTab('register')
                setError('')
              }}
            >
              注册
            </button>
          </div>

          <form onSubmit={onSubmit}>
            <VStack gap={3}>
              <TextInput
                label="手机号"
                value={account}
                onChange={(value) => setAccount(value)}
                placeholder="后端按手机号登记账号"
                autoComplete="username"
              />

              {tab === 'register' && (
                <TextInput
                  label="昵称（可选）"
                  value={nickname}
                  onChange={(value) => setNickname(value)}
                  placeholder="不填则默认用手机号"
                />
              )}

              <TextInput
                label="密码"
                type="password"
                value={password}
                onChange={(value) => setPassword(value)}
                placeholder="至少 6 位"
                autoComplete={tab === 'login' ? 'current-password' : 'new-password'}
              />

              {error !== '' && <p className="field-error">{error}</p>}

              <Button
                label={pending ? '处理中…' : tab === 'login' ? '登录' : '注册并登录'}
                type="submit"
                variant="primary"
                isDisabled={pending}
                width="100%"
              />
            </VStack>
          </form>

          <p className="hint-note">
            已对接 POST /api/v1/auth/login、/auth/register；登录成功后 token 存本地，后续请求带
            Authorization: Bearer。
          </p>
        </VStack>
      </Card>
    </div>
  )
}
