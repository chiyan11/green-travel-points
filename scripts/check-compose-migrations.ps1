# 校验 deploy/docker-compose.yml 是否挂载了 db/migrations 下全部 up 迁移。
#
# 背景：MySQL 只在数据目录为空时执行 /docker-entrypoint-initdb.d 里的脚本，
# 漏挂一个迁移，空卷起的库就少表（历史问题：000006/000007 曾漏挂，
# 导致取消订单、调账、配置变更报 "Table doesn't exist"）。
#
# 用法： pwsh -File .\scripts\check-compose-migrations.ps1
# 退出码 0 = 全部已挂载；1 = 有漏挂或文件缺失。

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$composePath = Join-Path $repoRoot 'deploy\docker-compose.yml'
$migrationsDir = Join-Path $repoRoot 'db\migrations'

if (-not (Test-Path $composePath)) { Write-Host "找不到 $composePath"; exit 1 }

$compose = Get-Content $composePath -Raw -Encoding UTF8
$mounted = [regex]::Matches($compose, 'db/migrations/([0-9A-Za-z_\-\.]+\.up\.sql)') |
  ForEach-Object { $_.Groups[1].Value }

$expected = Get-ChildItem $migrationsDir -Filter '*.up.sql' | Select-Object -ExpandProperty Name | Sort-Object
$mountedSet = $mounted | Sort-Object -Unique

$missing = @($expected | Where-Object { $mountedSet -notcontains $_ })
$extra = @($mountedSet | Where-Object { $expected -notcontains $_ })

if ($missing.Count -gt 0) {
  Write-Host "以下迁移未挂载到 compose，空卷起库会缺表：" -ForegroundColor Red
  $missing | ForEach-Object { Write-Host "  - $_" }
}
if ($extra.Count -gt 0) {
  Write-Host "compose 挂载了 db/migrations 里不存在的文件：" -ForegroundColor Red
  $extra | ForEach-Object { Write-Host "  - $_" }
}
if ($missing.Count -gt 0 -or $extra.Count -gt 0) { exit 1 }

Write-Host ("compose 迁移挂载完整：{0} / {1} 个 up 迁移" -f $mountedSet.Count, $expected.Count)
