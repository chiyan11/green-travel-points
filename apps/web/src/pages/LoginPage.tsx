import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { loginMock } from '@/services/auth'

/** 登录页（MOCK）：任意非空昵称+密码即可进入；联调后对接 POST /api/v1/auth/login。 */
export function LoginPage(): ReactNode {
  const navigate = useNavigate()
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (nickname.trim() === '' || password.trim() === '') {
      setError('请输入昵称和密码（原型阶段任意非空即可）')
      return
    }
    loginMock(nickname)
    navigate('/', { replace: true })
  }

  return (
    <div className="auth-wrap">
      <section className="hero">
        <h2>登录绿行积分</h2>
        <p>上报绿色出行，积攒碳积分，兑换绿色好物。</p>
      </section>
      <section className="card">
        <form onSubmit={onSubmit}>
          <div className="form-field">
            <label htmlFor="nickname">昵称 / 账号</label>
            <input
              id="nickname"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder="演示：输入任意昵称"
              autoComplete="username"
            />
          </div>
          <div className="form-field">
            <label htmlFor="password">密码</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="原型任意非空密码"
              autoComplete="current-password"
            />
          </div>
          {error && <p className="field-error">{error}</p>}
          <button type="submit" className="btn" style={{ width: '100%' }}>
            登录 / 进入原型
          </button>
        </form>
        <p className="mock-note" style={{ marginTop: 12 }}>
          MOCK：未对接后端认证，Token 仅存浏览器本地；注册/登录接口见接口需求清单 M1。
        </p>
      </section>
    </div>
  )
}
