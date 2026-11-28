# 部署冒烟：对着已经起好的 API 跑一遍「探针 → 登录 → 只读巡检 →（可选）写路径」。
#
# 默认**只读**：不发任何写请求，可以安全地对着任意环境的 API 跑。
# 需要验证「建商品 / 改库存 / 下单 / 取消 / 调账 / 改参数」这条写路径时，
# 显式加 -AllowMutate；此时脚本会写数据，只在本地一次性容器/演示库上使用。
#
# 写路径会留下什么、怎么收尾：
#   · 商品：结束时 PATCH 为 OFFLINE 并把名字改成「已归档冒烟商品-<stamp>」，
#     不再留下 `部署冒烟商品-*` 的可见残件（后端没有删除商品端点）；
#   · 积分：调账 -5 之后立刻 +5 冲正，结束前比对 1001 的 balance 与开跑前一致；
#   · 参数：daily_points_cap +20 再改回原值；
#   · 审计：audit_log 只追加，写路径必然新增记录（这是验收证据，不是污染）；
#     要“零写入”就跑默认只读模式。
#
# 共享库保护：ApiBase 不是本机回环地址时默认拒绝执行（共享库是全组数据）。
# 确实要对着远程环境跑，显式加 -AllowRemote（写路径还要再叠加 -AllowMutate）。
#
# 用法：
#   pwsh -File .\scripts\smoke-deploy.ps1                       # 只读，默认 http://127.0.0.1:8080
#   pwsh -File .\scripts\smoke-deploy.ps1 -ApiBase http://127.0.0.1:18080
#   pwsh -File .\scripts\smoke-deploy.ps1 -AllowMutate           # 本地库，跑完整写路径
#
# 前置：写路径与后台登录需要种子账号（db/seed/dev_demo.sql，compose 首次启动自动导入）：
#   1001 13800000001 / demo123456（CITIZEN）
#   1002 13800000002 / admin123456（ADMIN）
# 退出码 0 = 全部通过；1 = 有失败步骤或被共享库保护拦下。

param(
  [string]$ApiBase = 'http://127.0.0.1:8080',
  [string]$CitizenPhone = '13800000001',
  [string]$CitizenPassword = 'demo123456',
  [string]$AdminPhone = '13800000002',
  [string]$AdminPassword = 'admin123456',
  [switch]$AllowMutate,
  [switch]$AllowRemote
)

$ErrorActionPreference = 'Stop'
$failed = @()

$apiHost = ([uri]$ApiBase).Host
$loopbackHosts = @('127.0.0.1', 'localhost', '::1', '[::1]')
if ($loopbackHosts -notcontains $apiHost -and -not $AllowRemote) {
  Write-Host "拒绝执行：$ApiBase 不是本机回环地址。" -ForegroundColor Red
  Write-Host "共享库（见 docs/dev/shared-db.md）是全组数据，冒烟脚本默认不碰它；确需远程执行请显式加 -AllowRemote。" -ForegroundColor Yellow
  exit 1
}

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

Step '只读巡检（迁移表与种子数据是否真的在）'
$summaryBefore = Call 'GET' '/api/v1/points/summary' $null $ct
Check ($summaryBefore.status -eq 200) "GET /points/summary -> $($summaryBefore.status) balance=$($summaryBefore.data.data.balance)"
$balanceBefore = [int64]$summaryBefore.data.data.balance
$catalog = Call 'GET' '/api/v1/products?page=1&pageSize=10' $null $null
Check ($catalog.status -eq 200 -and $catalog.data.data.total -gt 0) "GET /products -> $($catalog.status) 共 $($catalog.data.data.total) 件（000005 种子商品）"
$adminProducts = Call 'GET' '/api/v1/admin/products?page=1&pageSize=10' $null $at
Check ($adminProducts.status -eq 200) "GET /admin/products -> $($adminProducts.status)"
$adminOrders = Call 'GET' '/api/v1/admin/orders?page=1&pageSize=10' $null $at
Check ($adminOrders.status -eq 200) "GET /admin/orders -> $($adminOrders.status)（000005 redeem_order）"
$auditBefore = Call 'GET' '/api/v1/admin/audit-logs?page=1&pageSize=5' $null $at
Check ($auditBefore.status -eq 200) "GET /admin/audit-logs -> $($auditBefore.status)（000007 audit_log）"
$auditTotalBefore = [int64]$auditBefore.data.data.total
$cfgBefore = Call 'GET' '/api/v1/admin/config' $null $at
Check ($cfgBefore.status -eq 200 -and $cfgBefore.data.data.Count -gt 0) "GET /admin/config -> $($cfgBefore.status) 参数 $($cfgBefore.data.data.Count) 项"

if (-not $AllowMutate) {
  Write-Host ""
  if ($failed.Count -gt 0) {
    Write-Host ("SMOKE FAILED（只读）：" + ($failed -join '; ')) -ForegroundColor Red
    exit 1
  }
  Write-Host "SMOKE OK（只读）：探针 / 登录 / 积分 / 商品 / 订单 / 审计 / 参数 均可用；未写入任何数据。" -ForegroundColor Green
  Write-Host "需要验证写路径（建商品 → 下单 → 取消 → 调账 → 改参数）时加 -AllowMutate。" -ForegroundColor Yellow
  exit 0
}

Write-Host ""
Write-Host "!! 写路径已开启（-AllowMutate）：将创建商品、下单并取消、调账 ±5、改系统参数后再还原。" -ForegroundColor Yellow

$stamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
$productId = $null
$skuId = $null
$adjustedDown = $false
$capValue = $null
$capBumped = $false

try {
  Step '后台：商品与库存'
  $prod = Call 'POST' '/api/v1/admin/products' @{
    merchantId = 1001
    name       = "部署冒烟商品-$stamp"
    category   = '冒烟'
    status     = 'ONLINE'
    skus       = @(@{ name = '标准款'; pointsPrice = 50; stock = 5 })
  } $at
  Check ($prod.status -eq 201 -and $prod.data.code -eq 'OK') "POST /admin/products -> $($prod.status) id=$($prod.data.data.id)"
  if ($prod.data.data.id) {
    $productId = $prod.data.data.id
    $skuId = $prod.data.data.skus[0].id
  }
  if ($skuId) {
    $stock = Call 'PUT' "/api/v1/admin/skus/$skuId/stock" @{ stock = 6 } $at
    Check ($stock.status -eq 200 -and $stock.data.data.stock -eq 6) "PUT /admin/skus/$skuId/stock -> $($stock.status) stock=$($stock.data.data.stock)"
  }

  Step '用户下单后取消（取消要写 point_batch_txn）'
  if ($productId) {
    $order = Call 'POST' '/api/v1/orders' @{ productId = $productId; skuId = $skuId; receiver = '部署冒烟' } $ct
    Check ($order.status -eq 201 -or $order.status -eq 200) "POST /orders -> $($order.status) id=$($order.data.data.id)"
    $orderId = $order.data.data.id
    if ($orderId) {
      $cancel = Call 'POST' "/api/v1/orders/$orderId/cancel" @{ reason = '部署冒烟：取消校验' } $ct
      Check ($cancel.status -eq 200 -and $cancel.data.code -eq 'OK') "POST /orders/$orderId/cancel -> $($cancel.status) status=$($cancel.data.data.status)"
      Check ($cancel.data.data.status -eq 'CANCELLED') "订单状态 -> $($cancel.data.data.status)"
    }
  } else {
    Check $false '商品创建失败，跳过后面的下单 / 取消步骤'
  }

  Step '后台：人工调账（要写 audit_log，结束冲正）'
  $adj = Call 'POST' '/api/v1/admin/points/adjust' @{ userId = 1001; amount = -5; reason = '部署冒烟：调账校验' } $at
  Check ($adj.status -eq 200 -and $adj.data.code -eq 'OK') "POST /admin/points/adjust -5 -> $($adj.status)"
  $adjustedDown = ($adj.status -eq 200 -and $adj.data.code -eq 'OK')

  Step '后台：系统参数变更（要写 audit_log，结束还原）'
  $capValue = ($cfgBefore.data.data | Where-Object { $_.key -eq 'daily_points_cap' }).value
  if ($capValue) {
    $bump = Call 'PUT' '/api/v1/admin/config' @{ items = @(@{ key = 'daily_points_cap'; value = "$([int]$capValue + 20)" }) } $at
    Check ($bump.status -eq 200 -and $bump.data.code -eq 'OK') "PUT /admin/config（+20）-> $($bump.status)"
    $capBumped = ($bump.status -eq 200 -and $bump.data.code -eq 'OK')
  } else {
    Check $false '没读到 daily_points_cap，跳过参数变更'
  }
} finally {
  Step '收尾（无论前面成败都要还原）'

  if ($adjustedDown) {
    $revert = Call 'POST' '/api/v1/admin/points/adjust' @{ userId = 1001; amount = 5; reason = '部署冒烟：冲正 -5' } $at
    Check ($revert.status -eq 200 -and $revert.data.code -eq 'OK') "POST /admin/points/adjust +5 冲正 -> $($revert.status)"
  }

  if ($capBumped -and $capValue) {
    $restore = Call 'PUT' '/api/v1/admin/config' @{ items = @(@{ key = 'daily_points_cap'; value = "$capValue" }) } $at
    Check ($restore.status -eq 200 -and $restore.data.code -eq 'OK') "PUT /admin/config（还原 $capValue）-> $($restore.status)"
  }

  if ($productId) {
    $archive = Call 'PATCH' "/api/v1/admin/products/$productId" @{ status = 'OFFLINE'; name = "已归档冒烟商品-$stamp" } $at
    Check ($archive.status -eq 200 -and $archive.data.data.status -eq 'OFFLINE') "PATCH /admin/products/$productId 下架归档 -> $($archive.status) status=$($archive.data.data.status)"
  }

  $summaryAfter = Call 'GET' '/api/v1/points/summary' $null $ct
  Check ($summaryAfter.status -eq 200) "GET /points/summary -> $($summaryAfter.status) balance=$($summaryAfter.data.data.balance)"
  Check ([int64]$summaryAfter.data.data.balance -eq $balanceBefore) "1001 余额回到运行前：$balanceBefore -> $($summaryAfter.data.data.balance)"

  $left = Call 'GET' '/api/v1/admin/products?page=1&pageSize=200' $null $at
  $residual = @($left.data.data.items | Where-Object { $_.name -like '部署冒烟商品-*' })
  Check ($residual.Count -eq 0) "无「部署冒烟商品-*」可见残留（本次运行新增：$(if ($residual.Count -eq 0) { '0' } else { $residual.Count }) 件）"

  $auditAfter = Call 'GET' '/api/v1/admin/audit-logs?page=1&pageSize=5' $null $at
  if ($auditAfter.status -eq 200) {
    Write-Host ("  审计追加：{0} -> {1} 条（audit_log 只追加，写路径必然新增，属验收证据）" -f $auditTotalBefore, $auditAfter.data.data.total)
  }
}

Write-Host ""
if ($failed.Count -gt 0) {
  Write-Host ("SMOKE FAILED: " + ($failed -join '; ')) -ForegroundColor Red
  exit 1
}
Write-Host "SMOKE OK：建商品 / 改库存 / 下单 / 取消 / 调账 / 改参数 / 查审计 全部通过，数据已还原。" -ForegroundColor Green
