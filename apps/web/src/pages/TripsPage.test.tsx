import { screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http } from 'msw'
import { useQuery } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import { TripsPage } from '@/pages/TripsPage'
import { getCarbonSeries } from '@/services/points'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { Trip } from '@/types/api'

function makeTrip(over: Partial<Trip> = {}): Trip {
  return {
    id: 'trip-1',
    mode: 'subway',
    distanceKm: 3.2,
    traveledAt: '2026-11-24T02:00:00Z',
    carbonSavedGrams: 1120,
    rawPoints: 40,
    grantedPoints: 40,
    cappedByDaily: false,
    status: 'VALID',
    ...over,
  }
}

function formOf(): HTMLFormElement {
  return screen.getByLabelText(/距离/).closest('form') as HTMLFormElement
}

async function submitTrip(distance: string) {
  const user = userEvent.setup()
  await user.clear(screen.getByLabelText(/距离/))
  await user.type(screen.getByLabelText(/距离/), distance)
  await user.click(within(formOf()).getByRole('button', { name: '上报行程' }))
  return user
}

describe('TripsPage', () => {
  it('上报成功后展示后端核算的碳减排与积分', async () => {
    let submittedBody: unknown = null
    server.use(
      http.get('*/api/v1/trips', () => okJson(pageOf([]))),
      http.post('*/api/v1/trips', async ({ request }) => {
        submittedBody = await request.json()
        return okJson(makeTrip())
      }),
    )
    renderPage(<TripsPage />)

    await submitTrip('3.2')

    expect(await screen.findByText('本次结果（后端核算）')).toBeInTheDocument()
    expect(screen.getByText('40 分')).toBeInTheDocument()
    // 请求体用后端字段名 traveledAt 发送，不是页面用的 startedAt。
    expect(submittedBody).toMatchObject({ mode: 'subway', distanceKm: 3.2 })
    expect(submittedBody).toHaveProperty('traveledAt')
  })

  it('同一笔出行重复提交时沿用后端返回的首次结果，不重复计分', async () => {
    let postCount = 0
    let listCalls = 0
    server.use(
      http.get('*/api/v1/trips', () => {
        listCalls += 1
        return okJson(pageOf(listCalls === 1 ? [] : [makeTrip()]))
      }),
      http.post('*/api/v1/trips', () => {
        postCount += 1
        // 后端幂等：第二次提交返回首条记录（HTTP 200 + 同一 id / 同一 grantedPoints）。
        return okJson(makeTrip())
      }),
    )
    renderPage(<TripsPage />)

    await submitTrip('3.2')
    await screen.findByText('本次结果（后端核算）')
    await submitTrip('3.2')

    await waitFor(() => expect(postCount).toBe(2))
    // 结果面板仍显示 40 分（不是 80），列表里也只有一条记录。
    expect(screen.getByText('40 分')).toBeInTheDocument()
    expect(screen.queryByText('80 分')).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('共 1 条')).toBeInTheDocument())
  })

  it('后端拒绝时把错误码文案展示在表单里', async () => {
    server.use(
      http.get('*/api/v1/trips', () => okJson(pageOf([]))),
      http.post('*/api/v1/trips', () =>
        failJson(400, 'INVALID_ARGUMENT', '行程距离超出上限 100km'),
      ),
    )
    renderPage(<TripsPage />)

    await submitTrip('3.2')

    expect(await screen.findByText('行程距离超出上限 100km')).toBeInTheDocument()
    expect(screen.queryByText('本次结果（后端核算）')).not.toBeInTheDocument()
  })

  it('上报成功后让 7 日碳趋势重新取数（首页/积分页不显示旧图）', async () => {
    let trendCalls = 0
    server.use(
      http.get('*/api/v1/trips', () => okJson(pageOf([]))),
      http.post('*/api/v1/trips', () => okJson(makeTrip())),
      http.get('*/api/v1/points/daily-carbon', () => {
        trendCalls += 1
        return okJson([{ date: '2026-11-24', grams: 1120 }])
      }),
    )

    // 复刻首页/积分页的查询键，验证提交后它会被失效并重新请求
    function TrendProbe() {
      const { dataUpdatedAt } = useQuery({ queryKey: ['carbon-trend-7d'], queryFn: () => getCarbonSeries(7) })
      return <span data-testid="trend-at">{dataUpdatedAt}</span>
    }

    renderPage(
      <>
        <TripsPage />
        <TrendProbe />
      </>,
    )

    await waitFor(() => expect(trendCalls).toBeGreaterThanOrEqual(1))
    const before = trendCalls

    await submitTrip('3.2')
    await screen.findByText('本次结果（后端核算）')

    await waitFor(() => expect(trendCalls).toBeGreaterThan(before))
  })

  it('行程列表加载失败时给出失败态，不伪装成「还没有行程记录」', async () => {
    server.use(
      http.get('*/api/v1/trips', () => failJson(500, 'INTERNAL', '查询超时')),
    )
    renderPage(<TripsPage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('查询超时')
    expect(screen.queryByText('还没有行程记录')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '重试' })).toBeEnabled()
  })
})
