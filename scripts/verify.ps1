# 一键校验：后端静态检查 + 单测 + 构建，前端类型检查 + 单测 + 构建。
# 用法： pwsh -File .\scripts\verify.ps1
# 任一环节失败即以非 0 退出，可直接用于本地自检或交接前门禁。

$ErrorActionPreference = 'Continue'
. (Join-Path $PSScriptRoot 'dev-env.ps1')

$repoRoot = Split-Path -Parent $PSScriptRoot
$failed = @()

function Step([string]$name, [scriptblock]$body) {
  Write-Host ""
  Write-Host "===== $name =====" -ForegroundColor Cyan
  try {
    & $body
    if ($LASTEXITCODE -ne 0 -and $null -ne $LASTEXITCODE) { throw "exit $LASTEXITCODE" }
  } catch {
    Write-Host "FAILED: $name -> $_" -ForegroundColor Red
    $script:failed += $name
  }
}

$api = Join-Path $repoRoot 'apps\api'
$web = Join-Path $repoRoot 'apps\web'

Step 'api: gofmt'     { Push-Location $api; try { $out = gofmt -l .; if ($out) { Write-Host $out; throw "以下文件未格式化" } } finally { Pop-Location } }
Step 'api: go vet'    { Push-Location $api; try { go vet ./... } finally { Pop-Location } }
Step 'api: go test'   { Push-Location $api; try { go test ./... -count=1 } finally { Pop-Location } }
Step 'api: go build'  { Push-Location $api; try { go build -o (Join-Path $repoRoot '.cache\bin\server.exe') ./cmd/server } finally { Pop-Location } }
Step 'web: api contract' { Push-Location $web; try { npm run gen:api; if ($LASTEXITCODE -ne 0) { throw "gen:api 执行失败" }; $dirty = git status --porcelain -- src/types/api.ts; if ($dirty) { Write-Host $dirty; throw "src/types/api.ts 与 docs/api/openapi.yaml 不同步，请提交重新生成的结果" } } finally { Pop-Location } }
Step 'web: tsc'       { Push-Location $web; try { npx --no-install tsc --noEmit } finally { Pop-Location } }
Step 'web: vitest'    { Push-Location $web; try { npx --no-install vitest run } finally { Pop-Location } }
Step 'web: build'     { Push-Location $web; try { npx --no-install vite build } finally { Pop-Location } }

Write-Host ""
if ($failed.Count -gt 0) {
  Write-Host ("VERIFY FAILED: " + ($failed -join ', ')) -ForegroundColor Red
  exit 1
}
Write-Host "VERIFY OK" -ForegroundColor Green
