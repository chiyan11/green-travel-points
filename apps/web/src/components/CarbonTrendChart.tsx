/**
 * 近 7 日碳减排趋势折线图（ECharts）。
 * 原型占位：数据来自 services/points#getRecentCarbonSeries，联调换真实聚合接口。
 */
import { useEffect, useRef, type ReactNode } from 'react'
import * as echarts from 'echarts'
import type { DailyCarbonPoint } from '@/services/points'

export function CarbonTrendChart({ data }: { data: DailyCarbonPoint[] }): ReactNode {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!ref.current) return
    const chart = echarts.init(ref.current)
    chart.setOption({
      grid: { left: 48, right: 16, top: 32, bottom: 28 },
      tooltip: { trigger: 'axis' },
      xAxis: {
        type: 'category',
        data: data.map((d) => d.date),
        axisLine: { lineStyle: { color: '#94a3b8' } },
      },
      yAxis: {
        type: 'value',
        name: 'gCO2e',
        splitLine: { lineStyle: { color: '#e2e8f0' } },
      },
      series: [
        {
          name: '碳减排',
          type: 'line',
          smooth: true,
          data: data.map((d) => d.grams),
          areaStyle: { opacity: 0.15 },
          lineStyle: { width: 2, color: '#16a34a' },
          itemStyle: { color: '#16a34a' },
        },
      ],
    })
    const ro = new ResizeObserver(() => chart.resize())
    ro.observe(ref.current)
    return () => {
      ro.disconnect()
      chart.dispose()
    }
  }, [data])

  return <div ref={ref} style={{ width: '100%', height: 280 }} role="img" aria-label="近 7 日碳减排趋势图" />
}
