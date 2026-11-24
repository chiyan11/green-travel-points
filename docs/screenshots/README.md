# E2E 验收截图

由 `apps/web/e2e/main-flow.spec.ts` 在真实前后端上跑出来，命令：

```powershell
cd apps/web
$env:E2E_API_TARGET = 'http://127.0.0.1:8080'   # 后端换端口时改这里
npm run test:e2e
```

用例每次注册一个新手机号，避免共享库里的历史数据影响断言；截图会覆盖同名文件，改主链路后重跑即可。

| 文件 | 场景 | 断言要点 |
| --- | --- | --- |
| `01-login.png` | 未登录访问 `/` | 重定向到登录页，展示「登录绿行积分」 |
| `02-register.png` | 切到注册页并填好表单 | 昵称字段只在注册态出现 |
| `03-home.png` | 注册后进入首页 | 主导航 6 项、首页视觉基线 |
| `04-trips.png` | 上报地铁 50 km | 结果面板给出碳减排 7.55 kg、发放 75 分 |
| `05-points.png` | 积分页 | 当前余额 150 分、近 7 日趋势、积分流水两条 |
| `06-mall.png` | 商城列表 | 在售商品列表与「我的余额」 |
| `07-product-detail.png` | 商品详情 | 库存、每人限购、收货人输入、确认兑换可点 |
| `08-orders-created.png` | 兑换后落到兑换记录 | 新订单状态「待支付」、可取消并返还 |
| `09-orders-cancelled.png` | 取消订单 | 状态变为「已取消」，积分返还 |
| `10-points-after-cancel.png` | 取消后回积分页 | 余额回到 150 分 |
| `11-login-error.png` | 错误密码登录 | 后端错误文案展示，不跳转首页 |

注：积分页的「近 7 日碳减排」当前仍是前端演示序列（P4 待办 #1），接上 `GET /api/v1/points/daily-carbon` 后需重拍 `05-points.png`。
