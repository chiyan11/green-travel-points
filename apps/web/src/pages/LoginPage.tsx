import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { IconLeaf, IconLock, IconPhone, IconUser } from '@/components/icons'
import { login, register } from '@/services/auth'
import { ApiError } from '@/lib/http'

type Tab = 'login' | 'register'

/**
 * 登录 / 注册页：左侧品牌区，右侧表单卡。
 *
 * 后端按手机号登记账号：注册可填昵称（省略则用手机号），登录用手机号 + 密码。
 * 文案固定「登录绿行积分」，E2E 用它确认首屏。
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
    <div className="auth">
      <aside className="auth__brand">
        <div className="auth__logo">
          <span className="auth__logo-mark">
            <IconLeaf size={22} />
          </span>
          绿行积分
        </div>

        <div>
          <p className="auth__claim">把每一次低碳出行，算成看得见的积分。</p>
          <p className="auth__claim-note">
            步行、骑行、地铁、公交都算数。攒下的积分可以在商城换绿色好物。
          </p>
        </div>

        <dl className="auth__facts">
          <div>
            <dd className="auth__fact-value">192 g</dd>
            <dt className="auth__fact-label">每公里基准排放</dt>
          </div>
          <div>
            <dd className="auth__fact-value">100 g = 1 分</dd>
            <dt className="auth__fact-label">碳减排折算积分</dt>
          </div>
          <div>
            <dd className="auth__fact-value">500 分</dd>
            <dt className="auth__fact-label">单日发放上限</dt>
          </div>
        </dl>
      </aside>

      <section className="auth__panel">
        <div className="auth__form">
          <div>
            <h1 className="auth__title">登录绿行积分</h1>
            <p className="auth__sub">上报绿色出行，积攒碳积分，兑换绿色好物。</p>
          </div>

          <div className="segmented" role="group" aria-label="登录或注册">
            <button
              type="button"
              className={tab === 'login' ? 'segmented__btn is-active' : 'segmented__btn'}
              onClick={() => {
                setTab('login')
                setError('')
              }}
            >
              登录
            </button>
            <button
              type="button"
              className={tab === 'register' ? 'segmented__btn is-active' : 'segmented__btn'}
              onClick={() => {
                setTab('register')
                setError('')
              }}
            >
              注册
            </button>
          </div>

          <form className="form" onSubmit={onSubmit}>
            <label className="field">
              <span className="field__label">手机号</span>
              <span className="input-wrap">
                <span className="input-icon">
                  <IconPhone size={18} />
                </span>
                <input
                  className="native-input input--icon"
                  value={account}
                  onChange={(e) => setAccount(e.target.value)}
                  inputMode="tel"
                  autoComplete="username"
                  placeholder="后端按手机号登记账号"
                />
              </span>
            </label>

            {tab === 'register' && (
              <label className="field">
                <span className="field__label">昵称（可选）</span>
                <span className="input-wrap">
                  <span className="input-icon">
                    <IconUser size={18} />
                  </span>
                  <input
                    className="native-input input--icon"
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    placeholder="不填就用手机号"
                  />
                </span>
              </label>
            )}

            <label className="field">
              <span className="field__label">密码</span>
              <span className="input-wrap">
                <span className="input-icon">
                  <IconLock size={18} />
                </span>
                <input
                  className="native-input input--icon"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete={tab === 'login' ? 'current-password' : 'new-password'}
                  placeholder="至少 6 位"
                />
              </span>
            </label>

            {error && <p className="field-error">{error}</p>}

            <button type="submit" className="btn btn--primary btn--block" disabled={pending}>
              {pending ? '提交中…' : tab === 'login' ? '登录' : '注册并登录'}
            </button>
          </form>
        </div>
      </section>
    </div>
  )
}
