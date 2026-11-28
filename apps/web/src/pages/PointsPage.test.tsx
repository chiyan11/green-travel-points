import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { PointsPage } from '@/pages/PointsPage'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { DailyCarbonPoint, PointTxn, PointsSummary } from '@/types/api'

function makeSummary(over: Partial<PointsSummary> = {}): PointsSummary {
  return {
    balance: 150,
    frozen: 0,
    available: 150,
    dailyCap: 500,
    earnedThisMonth: 150,
    earnedToday: 150,
    totalCarbonSavedGrams: 15100,
    ...over,
  }
}

function makeTxn(over: Partial<PointTxn> = {}): PointTxn {
  return {
    id: 1,
    amount: 75,
    balanceAfter: 150,
    createdAt: '2026-11-24T02:00:00Z',
    type: 'TRAVEL',
    reason: '',
    refType: 'trip_record',
    refId: 9,
    ...over,
  }
}

const trend: DailyCarbonPoint[] = [
  { date: '2026-11-23', grams: 640 },
  { date: '2026-11-24', grams: 7550 },
]

describe('PointsPage', () => {
  it('余额卡、趋势图、流水都拿到数据时正常渲染', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.get('*/api/v1/points/transactions', () => okJson(pageOf([makeTxn()]))),
      http.get('*/api/v1/points/daily-carbon', () => okJson(trend)),
    )
    renderPage(<PointsPage />)

    expect(await screen.findByText('出行获得')).toBeInTheDocument()
    expect(screen.getByText('150')).toBeInTheDocument()
  })

  it('积分流水加载失败时给出失败态与重试，不显示「还没有流水」', async () => {
    let calls = 0
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.get('*/api/v1/points/transactions', () => {
        calls += 1
        if (calls === 1) return failJson(500, 'INTERNAL', '流水查询失败')
        return okJson(pageOf([makeTxn()]))
      }),
      http.get('*/api/v1/points/daily-carbon', () => okJson(trend)),
    )
    renderPage(<PointsPage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('流水查询失败')
    expect(screen.queryByText('还没有流水')).not.toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: '重试' }))
    expect(await screen.findByText('出行获得')).toBeInTheDocument()
  })

  it('dailyCap 为 0 时显示「不限」，不画成 0 分上限', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary({ dailyCap: 0 }))),
      http.get('*/api/v1/points/transactions', () => okJson(pageOf([makeTxn()]))),
      http.get('*/api/v1/points/daily-carbon', () => okJson(trend)),
    )
    renderPage(<PointsPage />)

    expect(await screen.findByText('单日上限 不限')).toBeInTheDocument()
    expect(screen.getByText(/当前不设上限/)).toBeInTheDocument()
  })

  it('未知流水类型显示后端原始值，不硬塞成「人工调整」', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.get('*/api/v1/points/transactions', () =>
        okJson(pageOf([makeTxn({ id: 9, type: 'BONUS' as PointTxn['type'] })])),
      ),
      http.get('*/api/v1/points/daily-carbon', () => okJson(trend)),
    )
    renderPage(<PointsPage />)

    expect(await screen.findByText('BONUS')).toBeInTheDocument()
  })

  it('余额汇总加载失败时不把余额画成 0 分，而是给出失败态', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () => failJson(503, 'UNAVAILABLE', '账本服务不可用')),
      http.get('*/api/v1/points/transactions', () => okJson(pageOf([]))),
      http.get('*/api/v1/points/daily-carbon', () => okJson([])),
    )
    renderPage(<PointsPage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('账本服务不可用')
    expect(screen.queryByText('当前余额')).not.toBeInTheDocument()
  })
})
