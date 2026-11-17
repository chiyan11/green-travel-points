import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { fetchPing } from '@/lib/http'

const FLOW = ['上报行程', '计算碳减排量', '发放碳积分', '商城兑换']

const RULES = [
  { label: '基准排放因子', value: '192 gCO₂e/km', note: '燃油私家车人均口径' },
  { label: '地铁 / 公交', value: '41 / 68 gCO₂e/km', note: '来自 travel_mode 表' },
  { label: '步行 / 骑行 / 共享单车', value: '0 gCO₂e/km', note: '零直接排放' },
  { label: '积分兑换比例', value: '100 g = 1 分', note: '向下取整，单日上限 500 分' },
]

const ENTRIES = [
  { to: '/trips', emoji: '🚇', title: '上报行程', note: '步行/骑行/地铁/公交，一键算减排' },
  { to: '/points', emoji: '📈', title: '我的积分', note: '余额、流水与近 7 日趋势' },
  { to: '/mall', emoji: '🎁', title: '积分商城', note: '用碳积分兑换绿色好物' },
  { to: '/orders', emoji: '📦', title: '兑换记录', note: '订单状态与核销码' },
]

export function HomePage(): ReactNode {
  const { data, isPending, isError, error } = useQuery({
    queryKey: ['ping'],
    queryFn: fetchPing,
  })

  return (
    <>
      <section className="hero">
        <h2>把每一次低碳出行，算成看得见的积分</h2>
        <p>上报一次出行，系统核算碳减排量并折算积分，攒够就能在商城换好物。</p>
        <div className="hero__flow">
          {FLOW.map((step, i) => (
            <span key={step} style={{ display: 'contents' }}>
              <span className={i === FLOW.length - 1 ? 'flow-step flow-step--end' : 'flow-step'}>
                {step}
              </span>
              {i < FLOW.length - 1 && <span className="flow-arrow" aria-hidden="true">→</span>}
            </span>
          ))}
        </div>
      </section>

      <section className="grid">
        {ENTRIES.map((e) => (
          <Link to={e.to} className="card feature-card" key={e.to}>
            <div className="feature-card__icon" aria-hidden="true">{e.emoji}</div>
            <h3 className="feature-card__title">{e.title}</h3>
            <p className="feature-card__note">{e.note}</p>
          </Link>
        ))}
      </section>

      <section className="card">
        <div className="section-head">
          <h3>前后端连通性</h3>
          {data !== undefined && <span className="badge badge--ok">已连通</span>}
          {isError && <span className="badge badge--danger">未连接</span>}
        </div>
        {isPending && <p className="muted">正在请求 /api/v1/ping …</p>}
        {isError && (
          <p className="danger">
            请求失败：{error instanceof Error ? error.message : '未知错误'}
            <br />
            <span className="muted">
              请先启动后端 <code>go run ./cmd/server</code>，或检查 vite 代理目标端口。
            </span>
          </p>
        )}
        {data !== undefined && (
          <ul className="checklist">
            <li className="done">
              后端回复 <code>{data.pong}</code>
            </li>
            <li>运行环境：{data.env}</li>
          </ul>
        )}
      </section>

      <section>
        <div className="section-head">
          <h3>碳核算规则</h3>
          <span className="muted">口径与 travel_mode / sys_config 种子一致</span>
        </div>
        <div className="grid">
          {RULES.map((rule) => (
            <article className="card" key={rule.label}>
              <p className="stat-label muted" style={{ margin: 0 }}>{rule.label}</p>
              <p className="metric">{rule.value}</p>
              <p className="muted">{rule.note}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="card">
        <h3>当前进度</h3>
        <ul className="checklist">
          <li className="done">Sprint 0：Monorepo 骨架、领域规则单元测试、健康检查、迁移脚本</li>
          <li className="done">Sprint 1：用户注册登录、行程上报、碳减排与积分计算（已对接接口）</li>
          <li className="done">Sprint 2：积分账户与流水、商城浏览、兑换下单与状态机（已对接接口）</li>
          <li>Sprint 3：运营后台、报表、E2E 回归与实验报告定稿</li>
        </ul>
      </section>
    </>
  )
}
