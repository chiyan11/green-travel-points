# 部署冒烟：对着已经起好的 API 跑一遍「后台建商品 → 改库存 → 用户下单 → 取消 →
# 人工调账 → 改系统参数 → 查审计」，专门验证 000006 point_batch_txn、000007 audit_log
# 两张表真的存在（历史缺陷：compose 漏挂这两组迁移，空卷起的库一取消订单/调账/改配置就 500）。
#
# 用法（compose 起好后，或本地 go run 起后端后）：
#   pwsh -File .\scripts\smoke-deploy.ps1
#   pwsh -File .\scripts\smoke-deploy.ps1 -ApiBase http://127.0.0.1:18080
#
# 前置：库里要有可登录的演示账号（db/seed/dev_demo.sql）：
#   1001 13800000001 / demo123456（CITIZEN）
#   1002 13800000002 / admin123456（ADMIN）
# 退出码 0 = 全部通过；1 = 有失败步骤。

param(
  [string]$ApiBase = 'http://127.0.0.1:8080',
  [string]$CitizenPhone = '13800000001',
  [string]$CitizenPassword = 'demo123456',
  [string]$AdminPhone = '13800000002',
  [string]$AdminPassword = 'admin123456'
)

$ErrorActionPreference = 'Stop'
$failed = @()

function Call([string]$method, [string]$path, $body, [string]$token) {
  $headers = @{ 'Content-Type' = 'application/json' }
  if ($token) { $headers['Authorization'] = "Bearer $token" }
  if ($method -eq 'POST' -and $path -eq '/api/v1/orders') {
    $headers['Idempotency-Key'] = [guid]::NewGuid().ToString()
  }
  $uri = "$ApiBase$path"
  try {
    $resp = Invoke-WebRequest -Uri $uri -Method $method -Headers $headers -Body ($body | ConvertTo-Json -Compress -Depth 6) -TimeoutSec 30
    return @{ status = [int]$resp.StatusCode; data = ($resp.Content | ConvertFrom-Json) }
  } catch {
    $status = 0
    $data = $null
    if ($_.Exception.Response) {
      $status = [int]$_.Exception.Response.StatusCode
      $data = $_.ErrorDetails.Message | ConvertFrom-Json -ErrorAction SilentlyContinue
    }
    return @{ status = $status; data = $data; error = $_.Exception.Message }
  }
}

function Step([string]$name) { Write-Host ""; Write-Host "===== $name =====" -ForegroundColor Cyan }

function Check([bool]$ok, [string]$msg) {
  if ($ok) { Write-Host "  PASS  $msg" }
  else { Write-Host "  FAIL  $msg" -ForegroundColor Red; $script:failed += $msg }
}

Step '探针'
$h = Call 'GET' '/healthz' $null $null
Check ($h.status -eq 200 -and $h.data.code -eq 'OK') "GET /healthz -> $($h.status) $($h.data.code)"
$r = Call 'GET' '/readyz' $null $null
Check ($r.status -eq 200 -and $r.data.data.database -eq 'up') "GET /readyz -> $($r.status) database=$($r.data.data.database)"

Step '登录（公民 / 管理员）'
$cl = Call 'POST' '/api/v1/auth/login' @{ phone = $CitizenPhone; password = $CitizenPassword } $null
Check ($cl.status -eq 200 -and $cl.data.code -eq 'OK') "公民 $CitizenPhone 登录 -> $($cl.status)"
$ct = $cl.data.data.token
$al = Call 'POST' '/api/v1/auth/login' @{ phone = $AdminPhone; password = $AdminPassword } $null
Check ($al.status -eq 200 -and $al.data.code -eq 'OK') "管理员 $AdminPhone 登录 -> $($al.status)"
$at = $al.data.data.token

Step '后台：商品与库存（000005 / 000007 无审计写入前先跑通）'
$stamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
$prod = Call 'POST' '/api/v1/admin/products' @{
  merchantId = 1001
  name       = "部署冒烟商品-$stamp"
  category   = '冒烟'
  status     = 'ONLINE'
  skus       = @(@{ name = '标准款'; pointsPrice = 50; stock = 5 })
} $at
Check ($prod.status -eq 201 -and $prod.data.code -eq 'OK') "POST /admin/products -> $($prod.status) id=$($prod.data.data.id)"
$productId = $prod.data.data.id
$skuId = $prod.data.data.skus[0].id
$stock = Call 'PUT' "/api/v1/admin/skus/$skuId/stock" @{ stock = 6 } $at
Check ($stock.status -eq 200 -and $stock.data.data.stock -eq 6) "PUT /admin/skus/$skuId/stock -> $($stock.status) stock=$($stock.data.data.stock)"

Step '用户下单后取消（取消要写 point_batch_txn）'
$order = Call 'POST' '/api/v1/orders' @{ productId = $productId; skuId = $skuId; receiver = '部署冒烟' } $ct
Check ($order.status -eq 201 -or $order.status -eq 200) "POST /orders -> $($order.status) id=$($order.data.data.id)"
$orderId = $order.data.data.id
$cancel = Call 'POST' "/api/v1/orders/$orderId/cancel" @{ reason = '部署冒烟：取消校验' } $ct
Check ($cancel.status -eq 200 -and $cancel.data.code -eq 'OK') "POST /orders/$orderId/cancel -> $($cancel.status) status=$($cancel.data.data.status)"
Check ($cancel.data.data.status -eq 'CANCELLED') "订单状态 -> $($cancel.data.data.status)"

Step '后台：人工调账（要写 audit_log）'
$adj = Call 'POST' '/api/v1/admin/points/adjust' @{ userId = 1001; amount = -5; reason = '部署冒烟：调账校验' } $at
Check ($adj.status -eq 200 -and $adj.data.code -eq 'OK') "POST /admin/points/adjust -> $($adj.status)"

Step '后台：系统参数变更（要写 audit_log）'
$cfgBefore = Call 'GET' '/api/v1/admin/config' $null $at
Check ($cfgBefore.status -eq 200 -and $cfgBefore.data.data.Count -gt 0) "GET /admin/config -> $($cfgBefore.status) 参数 $($cfgBefore.data.data.Count) 项"
$cap = ($cfgBefore.data.data | Where-Object { $_.key -eq 'daily_points_cap' }).value
$bump = Call 'PUT' '/api/v1/admin/config' @{ items = @(@{ key = 'daily_points_cap'; value = "$([int]$cap + 20)" }) } $at
Check ($bump.status -eq 200 -and $bump.data.code -eq 'OK') "PUT /admin/config（+20）-> $($bump.status)"
$restore = Call 'PUT' '/api/v1/admin/config' @{ items = @(@{ key = 'daily_points_cap'; value = "$cap" }) } $at
Check ($restore.status -eq 200 -and $restore.data.code -eq 'OK') "PUT /admin/config（还原 $cap）-> $($restore.status)"

Step '后台：审计日志可查'
$audit = Call 'GET' '/api/v1/admin/audit-logs?page=1&pageSize=5' $null $at
Check ($audit.status -eq 200 -and $audit.data.data.total -gt 0) "GET /admin/audit-logs -> $($audit.status) 共 $($audit.data.data.total) 条"

Write-Host ""
if ($failed.Count -gt 0) {
  Write-Host ("SMOKE FAILED: " + ($failed -join '; ')) -ForegroundColor Red
  exit 1
}
Write-Host "SMOKE OK：建商品 / 改库存 / 下单 / 取消 / 调账 / 改参数 / 查审计 全部通过" -ForegroundColor Green
