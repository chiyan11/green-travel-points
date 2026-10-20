# 前端原型说明（P4，MOCK 版）

在 Sprint 0 骨架上补的用户端原型：页面用本地 MOCK 数据跑通"登录 → 上报行程 → 看积分 → 商城 → 兑换 → 订单"全流程，
后端接口就绪后只替换 `src/services/*`，页面与组件不动。

## 运行

```powershell
cd apps/web
npm install
npm run dev        # http://localhost:5173，任意非空昵称+密码即可进入
npm run test       # vitest，当前 19 例
npm run build      # tsc --noEmit + vite build
```

## 目录（新增部分）

```
src/
├── types/domain.ts        # 手写 DTO 与冻结常量（排放因子/100g=1分/日限500/距离0.1-100）
├── lib/
│   ├── validators.ts      # 距离校验、本地预览碳核算/取整/封顶（与后端 domain 同口径，带单测）
│   ├── orderStatus.ts     # 订单状态机元数据与合法跃迁
│   └── format.ts          # 积分/克数/时间格式化
├── mock/data.ts           # 原型内存数据仓库（联调后整体删除）
├── services/              # 对接层：auth/trips/points/products/orders（联调只改这里）
├── components/
│   ├── AppLayout.tsx      # 导航 + 后端徽标 + 会话/退出
│   └── RequireAuth.tsx    # 路由守卫（MOCK 会话）
└── pages/                 # Login/Trips/Points/Mall/ProductDetail/Orders/Profile + 原 Home
```

## 联调替换顺序

1. P1+P2 出 `docs/api/openapi.yaml` → `openapi-typescript` 生成类型，替换 `types/domain.ts` 的手写 DTO。
2. `src/services/*` 逐个换成 `api.get/post`（`lib/http.ts` 已封装 Envelope/ApiError），补 Authorization 头与 401 跳转。
3. 删除 `src/mock/data.ts`，页面 `queryFn` 改为真实接口。

## 与正式仓库合并提示

- `vitest.config.ts` 新增了 `@` 别名解析（原文件没有，否则纯函数测试无法解析 `@/` 导入）。
- 本目录的 `package-lock.json` 是本机新装生成；并入正式仓库时以仓库原 lockfile 为准，不要覆盖。
- 未改动后端、未改统一响应壳 `lib/http.ts` 的既有行为，原 5 个 http 单测保持通过。
