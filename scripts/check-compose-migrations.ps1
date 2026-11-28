# 校验 deploy/docker-compose.yml 是否把 db/migrations 下全部 up 迁移挂成了 initdb 脚本，
# 且 /docker-entrypoint-initdb.d 里的序号（01..N）连续、唯一。
#
# 背景：MySQL 只在数据目录为空时执行 /docker-entrypoint-initdb.d 里的脚本，
# 漏挂一个迁移，空卷起的库就少表（历史问题：000006/000007 曾漏挂，
# 导致取消订单、调账、配置变更报 "Table doesn't exist"）。
#
# 两条容易假绿的写法，这里都堵住：
#   1. 对全文正则会把注释行里的示例路径也算成“已挂载” → 只认以 `- ` 开头的真实挂载行，跳过 `#` 注释；
#   2. 只比文件名集合，注释掉一行仍报 11/11 → 还校验 initdb 目标序号连续唯一。
#
# 用法： pwsh -File .\scripts\check-compose-migrations.ps1
# 退出码 0 = 全部已挂载且序号正确；1 = 有漏挂 / 多余 / 序号问题 / 文件缺失。

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$composePath = Join-Path $repoRoot 'deploy\docker-compose.yml'
$migrationsDir = Join-Path $repoRoot 'db\migrations'

if (-not (Test-Path $composePath)) { Write-Host "找不到 $composePath"; exit 1 }
if (-not (Test-Path $migrationsDir)) { Write-Host "找不到 $migrationsDir"; exit 1 }

# 挂载行形如：  - ../db/migrations/000010_x.up.sql:/docker-entrypoint-initdb.d/10_x.sql:ro
$mountPattern = [regex]'^-+\s+[^#]*db/migrations/(?<file>[0-9A-Za-z_\-\.]+\.up\.sql):/docker-entrypoint-initdb\.d/(?<seq>[0-9]+)_.*\.sql'

$mounts = @()
$lineNo = 0
foreach ($line in (Get-Content $composePath -Encoding UTF8)) {
  $lineNo++
  $trimmed = $line.Trim()
  if ($trimmed -eq '' -or $trimmed.StartsWith('#')) { continue }
  $m = $mountPattern.Match($trimmed)
  if ($m.Success) {
    $mounts += [pscustomobject]@{
      File   = $m.Groups['file'].Value
      Seq    = [int]$m.Groups['seq'].Value
      Line   = $lineNo
      Source = $trimmed
    }
  }
}

$expected = @(Get-ChildItem $migrationsDir -Filter '*.up.sql' | Select-Object -ExpandProperty Name | Sort-Object)

$mountedNames = @($mounts | Select-Object -ExpandProperty File)
$missing = @($expected | Where-Object { $mountedNames -notcontains $_ })
$extra = @($mountedNames | Where-Object { $expected -notcontains $_ } | Sort-Object -Unique)

$problems = @()
foreach ($f in $missing) { $problems += "未挂载：$f" }
foreach ($f in $extra) { $problems += "挂载了 db/migrations 里不存在的文件：$f" }

# 同一文件挂了两次也算错（初始化会跑两遍）。
$dupNames = @($mounts | Group-Object File | Where-Object { $_.Count -gt 1 } | Select-Object -ExpandProperty Name)
foreach ($f in $dupNames) { $problems += "重复挂载：$f" }

# 序号必须从 1 开始连续、唯一，且与文件名里的迁移编号一致（000010 → 10）。
$seqs = @($mounts | Select-Object -ExpandProperty Seq | Sort-Object)
$dupSeqs = @($mounts | Group-Object Seq | Where-Object { $_.Count -gt 1 } | Select-Object -ExpandProperty Name)
foreach ($s in $dupSeqs) { $problems += "initdb 序号重复：$s" }
for ($i = 0; $i -lt $seqs.Count; $i++) {
  if ($seqs[$i] -ne ($i + 1)) {
    $problems += "initdb 序号不连续：期望 $($i + 1)，实际 $($seqs[$i])"
    break
  }
}
foreach ($m in $mounts) {
  $num = [int]([regex]::Match($m.File, '^(\d+)').Groups[1].Value)
  if ($num -ne $m.Seq) {
    $problems += "序号与文件名不一致：$($m.File) 挂在 $($m.Seq)_…（第 $($m.Line) 行）"
  }
}

if ($problems.Count -gt 0) {
  Write-Host "compose 迁移挂载检查未通过：" -ForegroundColor Red
  $problems | ForEach-Object { Write-Host "  - $_" }
  exit 1
}

Write-Host ("compose 迁移挂载完整：{0} / {1} 个 up 迁移，initdb 序号 1..{2} 连续唯一" -f `
    $mounts.Count, $expected.Count, $mounts.Count)
