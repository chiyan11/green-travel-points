# 一键校验：后端静态检查 + 单测 + 构建，前端类型检查 + 单测 + 构建。
# 用法： pwsh -File .\scripts\verify.ps1
# 任一环节失败即以非 0 退出，可直接用于本地自检或交接前门禁。
#
# 两个历史坑（N29）：
#   1. 每步都用 `$LASTEXITCODE` 判断失败，上一步的 0 会残留到本步，
#      命令不存在（native 命令只报非终止错误）时本步被误判为通过 → 每步先重置再判；
#   2. 依赖只写在文档里，PATH 里缺 go/npm 时结果是一串红字而不是明确结论 →
#      每步用 Assert-Command 先确认命令存在。

$ErrorActionPreference = 'Continue'
. (Join-Path $PSScriptRoot 'dev-env.ps1')

$repoRoot = Split-Path -Parent $PSScriptRoot
$failed = @()

function Assert-Command([string]$name) {
  $cmd = Get-Command $name -ErrorAction SilentlyContinue
  if ($null -eq $cmd) { throw "缺少命令：$name（请检查 PATH 或安装依赖）" }
  return $cmd
}

function Step([string]$name, [scriptblock]$body) {
  Write-Host ""
  Write-Host "===== $name =====" -ForegroundColor Cyan
  # 清掉上一步的退出码，避免 0 残留把本步的错误掩盖成通过。
  $global:LASTEXITCODE = 0
  try {
    & $body
    if ($LASTEXITCODE -ne 0) { throw "exit $LASTEXITCODE" }
  } catch {
    Write-Host "FAILED: $name -> $_" -ForegroundColor Red
    $script:failed += $name
  }
}

$api = Join-Path $repoRoot 'apps\api'
$web = Join-Path $repoRoot 'apps\web'

Step 'api: gofmt'     {
  Assert-Command gofmt | Out-Null
  Push-Location $api
  try {
    $out = gofmt -l .
    if ($LASTEXITCODE -ne 0) { throw "gofmt 退出码 $LASTEXITCODE" }
    if ($out) { Write-Host $out; throw "以下文件未格式化" }
    $global:LASTEXITCODE = 0
  } finally { Pop-Location }
}
Step 'api: go vet'    { Assert-Command go | Out-Null; Push-Location $api; try { go vet ./... } finally { Pop-Location } }
Step 'api: go test'   { Assert-Command go | Out-Null; Push-Location $api; try { go test ./... -count=1 } finally { Pop-Location } }
Step 'api: go build'  { Assert-Command go | Out-Null; Push-Location $api; try { go build -o (Join-Path $repoRoot '.cache\bin\server.exe') ./cmd/server } finally { Pop-Location } }
Step 'web: api contract' {
  Assert-Command npm | Out-Null
  Push-Location $web
  try {
    npm run gen:api
    if ($LASTEXITCODE -ne 0) { throw "gen:api 执行失败（exit $LASTEXITCODE）" }
    $dirty = git status --porcelain -- src/types/api.ts
    if ($LASTEXITCODE -ne 0) { throw "git status 执行失败（exit $LASTEXITCODE）" }
    if ($dirty) { Write-Host $dirty; throw "src/types/api.ts 与 docs/api/openapi.yaml 不同步，请提交重新生成的结果" }
  } finally { Pop-Location }
}
Step 'web: tsc'       { Assert-Command npx | Out-Null; Push-Location $web; try { npx --no-install tsc --noEmit } finally { Pop-Location } }
Step 'web: vitest'    { Assert-Command npx | Out-Null; Push-Location $web; try { npx --no-install vitest run } finally { Pop-Location } }
Step 'web: build'     { Assert-Command npx | Out-Null; Push-Location $web; try { npx --no-install vite build } finally { Pop-Location } }
Step 'deploy: compose migrations' {
  $pwsh = Assert-Command pwsh
  & $pwsh.Source -NoProfile -File (Join-Path $PSScriptRoot 'check-compose-migrations.ps1')
}

Write-Host ""
if ($failed.Count -gt 0) {
  Write-Host ("VERIFY FAILED: " + ($failed -join ', ')) -ForegroundColor Red
  exit 1
}
Write-Host "VERIFY OK" -ForegroundColor Green
