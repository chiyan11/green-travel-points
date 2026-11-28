import { screen } from '@testing-library/react'
import { http } from 'msw'
import { describe, expect, it } from 'vitest'
import { HomePage } from '@/pages/HomePage'
import { renderPage } from '@/test/render'
import { failJson, okJson, pageOf } from '@/test/http'
import { server } from '@/test/server'
import type { PointsSummary, Trip } from '@/types/api'

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

function makeTrip(over: Partial<Trip> = {}): Trip {
  return {
    id: 'trip-1',
    mode: 'subway',
    distanceKm: 50,
    traveledAt: '2026-11-24T02:00:00Z',
    carbonSavedGrams: 7550,
    rawPoints: 75,
    grantedPoints: 75,
    cappedByDaily: false,
    status: 'VALID',
    ...over,
  }
}

describe('HomePage', () => {
  it('最近行程取前三条并换成中文出行方式', async () => {
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.get('*/api/v1/trips', () => okJson(pageOf([makeTrip()]))),
    )
    renderPage(<HomePage />)

    expect(await screen.findByText(/地铁 · 50 km/)).toBeInTheDocument()
  })

  it('行程接口失败时「最近行程」显示失败态与重试，而不是空态', async () => {
    let calls = 0
    server.use(
      http.get('*/api/v1/points/summary', () => okJson(makeSummary())),
      http.get('*/api/v1/trips', () => {
        calls += 1
        if (calls === 1) return failJson(500, 'INTERNAL', '行程查询失败')
        return okJson(pageOf([makeTrip()]))
      }),
    )
    renderPage(<HomePage />)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('加载失败')
    expect(alert).toHaveTextContent('行程查询失败')
    expect(screen.queryByText('还没有行程记录')).not.toBeInTheDocument()
  })
})
