# 开发环境变量（Windows / PowerShell）
#
# 作用：把 Go 与 Node 的缓存目录重定向到仓库内的 .cache/。原因是部分受限环境
# （例如 DSH 文件沙箱只允许写工作区）无法写入 C:\Users\<用户> 下的默认缓存，
# 会直接 EPERM 失败。
#
# 用法（在当前 PowerShell 会话里点源，使变量生效）：
#   . .\scripts\dev-env.ps1
# 之后本会话内的 go / npm 命令都会使用仓库内缓存。
# 团队成员本机没有沙箱限制时可以不点源，直接用系统默认缓存。

$repoRoot = Split-Path -Parent $PSScriptRoot
$cacheRoot = Join-Path $repoRoot '.cache'

$env:GOPATH = Join-Path $cacheRoot 'go'
$env:GOMODCACHE = Join-Path $cacheRoot 'go\pkg\mod'
$env:GOCACHE = Join-Path $cacheRoot 'go-build'
$env:GOBIN = Join-Path $cacheRoot 'go\bin'
$env:GOTMPDIR = Join-Path $cacheRoot 'go-tmp'
$env:npm_config_cache = Join-Path $cacheRoot 'npm'

foreach ($p in @($cacheRoot, $env:GOPATH, $env:GOCACHE, $env:GOBIN, $env:GOTMPDIR, $env:npm_config_cache)) {
  if (-not (Test-Path $p)) { New-Item -ItemType Directory -Force -Path $p | Out-Null }
}

Write-Host "[dev-env] repo      = $repoRoot"
Write-Host "[dev-env] GOMODCACHE= $env:GOMODCACHE"
Write-Host "[dev-env] GOCACHE   = $env:GOCACHE"
Write-Host "[dev-env] npm cache = $env:npm_config_cache"
