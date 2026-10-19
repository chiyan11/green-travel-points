import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/components/AppLayout'
import { HomePage } from '@/pages/HomePage'
import { PlaceholderPage } from '@/pages/PlaceholderPage'

/**
 * 路由骨架 = 实验指导书要求的用户端核心流程：
 * 行程上报 → 积分查询 → 商城浏览 → 兑换记录 → 个人中心。
 * 每个占位路由都标注了归属 Sprint，Sprint 1 起逐页替换为真实实现。
 */
export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      {
        path: 'trips',
        element: (
          <PlaceholderPage
            title="行程上报"
            sprint="Sprint 1"
            summary="用户提交一次出行记录（模拟定位与距离），系统按出行方式核算碳减排量并发放积分。"
            tasks={[
              'POST /api/v1/trips：校验距离区间、出行方式、重复上报（同用户同时间段幂等）',
              '表单：出行方式下拉（步/骑/地铁/公交/共享单车）+ 距离输入 + 时间选择',
              '提交成功展示本次碳减排量、原始积分、实发积分与单日上限削减原因',
              '行程列表：分页 + 状态筛选（待校验/已发放/已驳回）',
            ]}
          />
        ),
      },
      {
        path: 'points',
        element: (
          <PlaceholderPage
            title="我的积分"
            sprint="Sprint 2"
            summary="查看积分余额、累计碳减排量与积分流水；积分只通过账本流水变更，不做余额直接覆盖。"
            tasks={[
              'GET /api/v1/points/summary：余额、累计减排、本月获得',
              'GET /api/v1/points/transactions：流水（获得/消耗/过期/撤销）分页',
              '图表：近 7 日 / 近 30 日碳减排趋势',
              '并发安全：账本写入使用乐观锁（version）或行锁扣减',
            ]}
          />
        ),
      },
      {
        path: 'mall',
        element: (
          <PlaceholderPage
            title="积分商城"
            sprint="Sprint 2"
            summary="浏览可兑换商品、查看所需积分与库存，提交兑换申请。"
            tasks={[
              'GET /api/v1/products：分页 + 分类筛选 + 库存标记',
              'GET /api/v1/products/{id}：详情、兑换规则、库存',
              '兑换预校验：积分余额、库存、单用户限兑数量',
              '下单：生成兑换单 + 冻结积分 + 发件箱事件',
            ]}
          />
        ),
      },
      {
        path: 'orders',
        element: (
          <PlaceholderPage
            title="兑换记录"
            sprint="Sprint 2"
            summary="兑换单状态机：待支付/处理中/待发货/已发货/已完成/已取消，支持取消回滚积分。"
            tasks={[
              'GET /api/v1/orders：按状态分页查询',
              '取消订单：释放冻结积分并写入冲正流水',
              '状态流转图与非法流转拦截（状态机单元测试）',
              '超时未支付自动取消（Outbox Worker 延迟消息）',
            ]}
          />
        ),
      },
      {
        path: 'profile',
        element: (
          <PlaceholderPage
            title="个人中心"
            sprint="Sprint 1"
            summary="账号信息、碳减排成就徽章与数据导出。"
            tasks={[
              '注册 / 登录（JWT）与个人信息维护',
              '碳减排成就：累计减排量里程碑徽章',
              '个人碳足迹月度报表导出（CSV）',
            ]}
          />
        ),
      },
    ],
  },
])
