<#
.SYNOPSIS
  自动化代码审查的“确定性基线”步骤：拉取最新代码、跑门禁、收集变更与依赖审计，
  生成一份供人或 AI 继续审查的报告。

.DESCRIPTION
  该脚本只做可重复、可验证的机械检查，不替代人工/AI 的语义审查：
    1. git fetch origin <BaseBranch>（默认只 fetch，不改本地 HEAD）；
       显式给 -AutoPull 且工作区干净、当前就在 BaseBranch 时才会 `git merge --ff-only`；
    2. 运行 scripts/verify.ps1（gofmt / vet / test / build / tsc / vitest / vite build / compose 迁移）；
    3. 运行 npm audit --omit=dev，记录生产依赖漏洞；
    4. 计算距上次审查的变更文件，写入 .cache/review/latest.md 与 latest.json；
    5. 读取 docs/项目分工与迭代计划.md，为每条任务标注负责人；文档中没有明确
       姓名依据的写“未分配”，并在报告中单独列出，不猜测、不编造姓名。

  设计约束（与仓库 AGENTS.md 一致）：
    - 不修改业务代码、不提交、不推送；
    - 工作区不干净时不 pull，只记录，避免把别人的改动混进审查；
    - 报告目录在 .cache/ 下，已被 .gitignore 忽略。

.PARAMETER RepoRoot
  仓库根目录，默认是脚本所在目录的上一级。

.PARAMETER BaseBranch
  默认分支，默认 main。

.PARAMETER ReportDir
  报告目录（相对仓库根），默认 .cache/review。

.PARAMETER SkipFetch
  跳过 git fetch（离线或本地验证脚本时用）。

.PARAMETER AutoPull
  允许在 BaseBranch 且工作区干净时 `git merge --ff-only origin/<BaseBranch>`。
  默认不合并：审查脚本不应悄悄移动别人的基线，要拉代码请显式打开这个开关。

.PARAMETER SkipVerify
  跳过 verify.ps1（只验证脚本本身时用）。

.PARAMETER SkipAudit
  跳过 npm audit（无网络时用）。

.PARAMETER MarkReviewed
  语义审查完成后，把当前 HEAD 写入 .cache/review/last-reviewed.txt。
  默认不写：脚本只做机械基线，不能替人或 AI 宣布“已审查”。
  开启后必须先通过 scripts/verify.ps1（-SkipVerify 时直接拒绝标记）。

.EXAMPLE
  pwsh -File .\scripts\auto-review.ps1

.EXAMPLE
  pwsh -File .\scripts\auto-review.ps1 -SkipFetch -SkipVerify -SkipAudit
#>
[CmdletBinding()]
param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path,
    [string]$BaseBranch = 'main',
    [string]$ReportDir = '.cache/review',
    [string]$AssignmentDoc = 'docs/项目分工与迭代计划.md',
    [switch]$SkipFetch,
    [switch]$AutoPull,
    [switch]$SkipVerify,
    [switch]$SkipAudit,
    [switch]$MarkReviewed
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Step([string]$Message) {
    Write-Host "==> $Message" -ForegroundColor Cyan
}

# 读取分工文档。只认该文档中的角色/姓名/模块/迭代表，不在脚本里写死姓名。
function Read-AssignmentSource {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "找不到分工文档：$Path"
    }

    $roles = @{}
    $names = @{}
    $modules = @()
    $sprintTasks = @{}
    $sprintColumns = @()
    $section = ''

    foreach ($raw in Get-Content -LiteralPath $Path -Encoding utf8) {
        $line = $raw.Trim()
        if ($line -match '^##\s+3\.') { $section = 'modules'; continue }
        if ($line -match '^##\s+4\.') { $section = 'sprints'; continue }
        if ($line -match '^###\s+4\.1') { $section = 'sprintTasks'; continue }
        if ($line -match '^###\s+4\.2') { $section = ''; continue }
        if ($line -match '^##\s+') { $section = ''; continue }

        # §1 角色总览：| **P1** | 角色名 | 职责 | 答辩要求 |
        if ($line -match '^\|\s*\*\*(P\d)\*\*\s*\|\s*([^|]+)\|') {
            $roles[$Matches[1]] = $Matches[2].Trim()
            continue
        }

        # §1 姓名表：| P1 | ______（待填写） | 技术栈 |
        if ($line -match '^\|\s*(P\d)\s*\|\s*([^|]*)\|') {
            $roleCode = $Matches[1]
            $name = $Matches[2].Trim()
            if ($name -match '^[_\-—\s]+$' -or $name -match '待填写') { $name = '' }
            $names[$roleCode] = $name
            continue
        }

        # §3 WBS：| M1 | 名称 | 主责 | 协作 | 实验 | 验收 |
        if ($section -eq 'modules' -and $line -match '^\|\s*(M\d+)\s*\|') {
            $cells = @($line.Trim('|') -split '\|' | ForEach-Object { $_.Trim() })
            if ($cells.Count -ge 6) {
                $modules += [pscustomobject]@{
                    Code       = $cells[0]
                    Name       = $cells[1]
                    Owner      = $cells[2]
                    Collab     = $cells[3]
                    Experiment = $cells[4]
                    Acceptance = $cells[5]
                }
            }
            continue
        }

        # §4.1 逐 Sprint 分工：| Sprint | P1 | P2 | P3 | P4 | P5 |
        if ($section -eq 'sprintTasks') {
            if ($line -match '^\|\s*Sprint\s*\|') {
                $sprintColumns = @($line.Trim('|') -split '\|' | ForEach-Object { $_.Trim() })
                continue
            }
            if ($line -match '^\|\s*(S\d+)\s*\|') {
                $cells = @($line.Trim('|') -split '\|' | ForEach-Object { $_.Trim() })
                $map = [ordered]@{}
                for ($i = 1; $i -lt $sprintColumns.Count -and $i -lt $cells.Count; $i++) {
                    $map[$sprintColumns[$i]] = $cells[$i]
                }
                $sprintTasks[$Matches[1]] = $map
            }
        }
    }

    return [pscustomobject]@{
        Roles       = $roles
        Names       = $names
        Modules     = $modules
        SprintTasks = $sprintTasks
    }
}

# 仅做“路径 → 分工文档模块”的保守映射；匹配不到就返回 $null，由调用方标未分配。
function Get-ModuleForPath {
    param([string]$Path, [object[]]$Modules)
    $p = $Path -replace '\\', '/'
    $rules = @(
        @{ Code = 'M12'; Pattern = '(^|/)([^/]+_test\.go|e2e/|test/)' },
        @{ Code = 'M13'; Pattern = '(^|/)deploy/|(^|/)\.github/|Dockerfile|nginx\.conf' },
        @{ Code = 'M11'; Pattern = '^docs/uml/|^docs/(设计规格|需求规格)' },
        @{ Code = 'M14'; Pattern = '^docs/(实验报告|代码审查)|^README\.md$' },
        @{ Code = 'M8';  Pattern = '^apps/web/src/pages/admin/|^apps/api/internal/(repo|httpserver)/admin\.go$' },
        @{ Code = 'M9';  Pattern = '^apps/web/src/(services|lib|types|themes)/|^apps/web/src/(routes\.tsx|main\.tsx)|^apps/web/(package\.json|vite\.config|tsconfig)|^docs/api/' },
        @{ Code = 'M10'; Pattern = '^apps/web/src/pages/' },
        @{ Code = 'M4';  Pattern = '^apps/api/internal/(eventbus|repo/outbox|httpserver/.*event)' },
        @{ Code = 'M7';  Pattern = '^apps/api/internal/(repo|httpserver)/orders|^apps/api/internal/domain/order' },
        @{ Code = 'M6';  Pattern = '^apps/api/internal/(repo|httpserver)/products|^apps/api/internal/domain/product' },
        @{ Code = 'M3';  Pattern = '^apps/api/internal/(repo/batches|domain/points|repo/points|httpserver/points)' },
        @{ Code = 'M2';  Pattern = '^apps/api/internal/(repo|httpserver|domain)/(trips|carbon)|^apps/web/src/(pages/TripsPage|services/trips)' },
        @{ Code = 'M1';  Pattern = '^apps/api/internal/httpserver/(auth|password)|^apps/web/src/(pages/LoginPage|services/auth|lib/session)' }
    )

    foreach ($rule in $rules) {
        if ($p -match $rule.Pattern) {
            $hit = @($Modules | Where-Object { $_.Code -eq $rule.Code })
            if ($hit.Count -gt 0) { return $hit[0] }
        }
    }
    return $null
}

# 把分工文档里的角色串（如 P2+P3）解析成姓名；姓名缺失时明确返回“未分配”。
function Resolve-Owner {
    param([string]$RoleText, [hashtable]$Names)
    $roles = @($RoleText -split '\+' | ForEach-Object { $_.Trim() } | Where-Object { $_ })
    $matched = @()
    foreach ($r in $roles) {
        if ($Names.ContainsKey($r) -and $Names[$r]) {
            $matched += [string]$Names[$r]
        }
    }
    if ($matched.Count -gt 0) { return ($matched -join '、') }
    return '未分配'
}

function Invoke-Git {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$GitArgs)
    $output = & git -C $RepoRoot @GitArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "git $($GitArgs -join ' ') 失败（exit $LASTEXITCODE）：$($output -join "`n")"
    }
    return @($output)
}

if (-not (Test-Path -LiteralPath (Join-Path $RepoRoot '.git'))) {
    throw "RepoRoot 不是 Git 仓库：$RepoRoot"
}
if (-not (Test-Path -LiteralPath (Join-Path $RepoRoot 'scripts/verify.ps1'))) {
    throw "找不到 scripts/verify.ps1：$RepoRoot"
}

$reportRoot = Join-Path $RepoRoot $ReportDir
New-Item -ItemType Directory -Force -Path $reportRoot | Out-Null
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$latestMarkdown = Join-Path $reportRoot 'latest.md'
$latestJson = Join-Path $reportRoot 'latest.json'
$verifyLog = Join-Path $reportRoot "verify-$stamp.log"
$lastReviewedFile = Join-Path $reportRoot 'last-reviewed.txt'

$startedAt = (Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz')
$status = 'PASS'
$fetchNote = '跳过（-SkipFetch）'
$mergeNote = '未执行'
$verifyExit = 0
$auditSummary = '未执行'
$auditCritical = 0
$auditHigh = 0

Write-Step "仓库：$RepoRoot"
$branch = (Invoke-Git rev-parse --abbrev-ref HEAD | Select-Object -First 1).Trim()
$headBefore = (Invoke-Git rev-parse HEAD | Select-Object -First 1).Trim()
$dirty = @(Invoke-Git status --porcelain)
$dirtyCount = $dirty.Count

if (-not $SkipFetch) {
    Write-Step "git fetch origin $BaseBranch"
    $fetchOutput = & git -C $RepoRoot fetch origin $BaseBranch 2>&1
    $fetchExit = $LASTEXITCODE
    if ($fetchExit -ne 0) {
        $fetchNote = "失败（exit $fetchExit）：$($fetchOutput -join ' ')"
        $status = 'FETCH_FAILED'
    } else {
        $fetchNote = '成功'
        if (-not $AutoPull) {
            $mergeNote = '未执行（默认只 fetch；需要拉代码请加 -AutoPull）'
        } elseif ($branch -eq $BaseBranch -and $dirtyCount -eq 0) {
            Write-Step "工作区干净，快进到 origin/$BaseBranch"
            $mergeOutput = & git -C $RepoRoot merge --ff-only "origin/$BaseBranch" 2>&1
            if ($LASTEXITCODE -eq 0) {
                $mergeNote = '已快进'
            } else {
                $mergeNote = "未快进（exit $LASTEXITCODE）：$($mergeOutput -join ' ')"
                if ($status -eq 'PASS') { $status = 'PULL_FAILED' }
            }
        } else {
            $mergeNote = if ($branch -ne $BaseBranch) {
                "跳过：当前在 $branch，不在 $BaseBranch"
            } else {
                "跳过：工作区有 $dirtyCount 个未提交改动，先不 pull，避免混入审查"
            }
        }
    }
}

$head = (Invoke-Git rev-parse HEAD | Select-Object -First 1).Trim()
$shortHead = $head.Substring(0, [Math]::Min(12, $head.Length))

# -MarkReviewed 只记录“语义审查已完成”的 HEAD，不覆盖 latest.md，
# 但必须先跑门禁：没过 verify 的 HEAD 不允许被标成已审查。
if ($MarkReviewed) {
    if ($SkipVerify) {
        Write-Host "REVIEW_MARK_REFUSED=-SkipVerify 时不能标记已审查" -ForegroundColor Red
        exit 2
    }
    Write-Step '运行门禁（-MarkReviewed 前置）'
    $pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
    if ($null -eq $pwsh) {
        Write-Host "REVIEW_MARK_REFUSED=找不到 pwsh，无法运行 verify.ps1" -ForegroundColor Red
        exit 2
    }
    & $pwsh.Source -NoProfile -File (Join-Path $RepoRoot 'scripts/verify.ps1')
    if ($LASTEXITCODE -ne 0) {
        Write-Host "REVIEW_MARK_REFUSED=verify.ps1 exit $LASTEXITCODE" -ForegroundColor Red
        exit 1
    }
    Set-Content -LiteralPath $lastReviewedFile -Value $head -Encoding ascii
    Write-Host "REVIEW_MARKED=$head"
    exit 0
}
$originHead = ''
try {
    $originHead = (Invoke-Git rev-parse "origin/$BaseBranch" | Select-Object -First 1).Trim()
} catch {
    $originHead = 'unknown'
}

# 变更范围：优先用上次审查的 SHA；没有就对比 origin/<BaseBranch>。
$range = "origin/$BaseBranch...HEAD"
$lastReviewed = ''
if (Test-Path -LiteralPath $lastReviewedFile) {
    $candidate = (Get-Content -LiteralPath $lastReviewedFile -Raw).Trim()
    if ($candidate -match '^[0-9a-f]{7,40}$') {
        & git -C $RepoRoot cat-file -e "$candidate^{commit}" 2>$null
        if ($LASTEXITCODE -eq 0) {
            $lastReviewed = $candidate
            $range = "$candidate..HEAD"
        }
    }
}

$changedFiles = @()
$diffStat = ''
try {
    $changedFiles = @(Invoke-Git diff --name-status $range)
    $diffStat = (Invoke-Git diff --stat $range) -join "`n"
} catch {
    $changedFiles = @("（无法计算变更范围 $range：$($_.Exception.Message)）")
}

# 按人分派：唯一依据是分工文档；姓名栏为空就写“未分配”，不猜姓名。
$assignmentPath = Join-Path $RepoRoot $AssignmentDoc
$assignmentSource = Read-AssignmentSource -Path $assignmentPath
$assignmentRecords = @()
$reviewTasks = @(
    '正确性审查：边界条件、并发/异步、事务与幂等、时间/时区口径。',
    '安全审查：认证与授权、注入、敏感信息、依赖与部署默认值。',
    '可维护性审查：命名、分层依赖、重复、注释与文档一致性。',
    '性能审查：明显低效查询、无上限分页、首屏包体。',
    '实验指导书对照：需求/设计/原型/测试/总结五章、UML、Git 协作、AI 使用记录、积分核心测试。'
)
foreach ($text in $reviewTasks) {
    $assignmentRecords += [pscustomobject]@{
        Task   = $text
        Module = '（审查维度）'
        Role   = '—'
        Owner  = '未分配'
        Basis  = '审查维度，未绑定具体模块/迭代'
    }
}

foreach ($m in $assignmentSource.Modules) {
    $assignmentRecords += [pscustomobject]@{
        Task   = "模块：$($m.Code) $($m.Name)"
        Module = $m.Code
        Role   = $m.Owner
        Owner  = (Resolve-Owner -RoleText $m.Owner -Names $assignmentSource.Names)
        Basis  = "§3 $($m.Code) 主责=$($m.Owner)"
    }
}

foreach ($entry in $changedFiles) {
    $text = [string]$entry
    if ($text -notmatch '^[A-Z?]\s+(.+)$') { continue }
    $path = $Matches[1].Trim()
    $module = Get-ModuleForPath -Path $path -Modules $assignmentSource.Modules
    if ($null -eq $module) {
        $assignmentRecords += [pscustomobject]@{
            Task   = "审查变更：$path"
            Module = '未匹配'
            Role   = '—'
            Owner  = '未分配'
            Basis  = '路径未命中分工文档 §3 的模块'
        }
        continue
    }

    $owner = Resolve-Owner -RoleText $module.Owner -Names $assignmentSource.Names
    $assignmentRecords += [pscustomobject]@{
        Task   = "审查变更：$path"
        Module = "$($module.Code) $($module.Name)"
        Role   = $module.Owner
        Owner  = $owner
        Basis  = "§3 $($module.Code) 主责=$($module.Owner)"
    }
}

$namedCount = @($assignmentSource.Names.Values | Where-Object { $_ }).Count
$assignmentWarning = if ($namedCount -eq 0) {
    '分工文档 §1 姓名栏为空：模块角色可匹配，但具体人名无法落位，全部按“未分配”处理。'
} else {
    ''
}
$unassignedRecords = @($assignmentRecords | Where-Object { $_.Owner -eq '未分配' })

$assignmentLines = [System.Collections.Generic.List[string]]::new()
$assignmentLines.Add('## 任务分派（按人）')
$assignmentLines.Add('')
$assignmentLines.Add('> 唯一依据：`docs/项目分工与迭代计划.md`。按模块主责匹配角色；文档没有明确姓名依据时写“未分配”，不猜测、不编造姓名。')
if ($assignmentWarning) {
    $assignmentLines.Add('')
    $assignmentLines.Add("> **注意**：$assignmentWarning")
}
$assignmentLines.Add('')
$assignmentLines.Add('| 任务 | 模块/迭代 | 匹配角色 | 负责人 | 依据 |')
$assignmentLines.Add('| --- | --- | --- | --- | --- |')
foreach ($r in $assignmentRecords) {
    $assignmentLines.Add("| $($r.Task) | $($r.Module) | $($r.Role) | $($r.Owner) | $($r.Basis) |")
}
$assignmentLines.Add('')
$assignmentLines.Add('### 未分配任务')
if ($unassignedRecords.Count -eq 0) {
    $assignmentLines.Add('- 无')
} else {
    foreach ($r in $unassignedRecords) {
        $assignmentLines.Add("- $($r.Task)（$($r.Basis)）")
    }
}
$assignmentBlock = $assignmentLines -join "`n"

if (-not $SkipVerify) {
    Write-Step '运行 scripts/verify.ps1（可能需要几分钟）'
    $verifyOutput = & pwsh -NoProfile -File (Join-Path $RepoRoot 'scripts/verify.ps1') 2>&1 |
        Tee-Object -FilePath $verifyLog
    $verifyExit = $LASTEXITCODE
    if ($verifyExit -ne 0) {
        $status = 'VERIFY_FAILED'
    }
} else {
    Write-Step '跳过 verify.ps1（-SkipVerify）'
    Set-Content -LiteralPath $verifyLog -Value 'skipped' -Encoding utf8
}

if (-not $SkipAudit) {
    Write-Step '运行 npm audit --omit=dev'
    Push-Location (Join-Path $RepoRoot 'apps/web')
    try {
        $auditRaw = & npm audit --omit=dev --json 2>$null
        $auditExit = $LASTEXITCODE
        if ($auditRaw) {
            try {
                $audit = $auditRaw | ConvertFrom-Json
                if ($audit.metadata -and $audit.metadata.vulnerabilities) {
                    $v = $audit.metadata.vulnerabilities
                    $auditCritical = [int]$v.critical
                    $auditHigh = [int]$v.high
                    $auditSummary = "critical=$($v.critical), high=$($v.high), moderate=$($v.moderate), low=$($v.low), total=$($v.total)"
                } else {
                    $auditSummary = "npm audit 无 vulnerabilities 字段（exit $auditExit）"
                }
            } catch {
                $auditSummary = "npm audit 输出无法解析（exit $auditExit）"
            }
        } else {
            $auditSummary = "npm audit 无输出（exit $auditExit）"
        }
    } catch {
        $auditSummary = "npm audit 执行失败：$($_.Exception.Message)"
    } finally {
        Pop-Location
    }
} else {
    Write-Step '跳过 npm audit（-SkipAudit）'
    $auditSummary = '跳过（-SkipAudit）'
}

if ($auditCritical -gt 0) {
    # 不覆盖已经出现的失败状态：门禁没过比依赖漏洞更根本，退出码要保留 VERIFY_FAILED。
    if ($status -eq 'PASS') { $status = 'SECURITY_CRITICAL' }
}

$verifyTail = ''
if (Test-Path -LiteralPath $verifyLog) {
    $verifyTail = (Get-Content -LiteralPath $verifyLog -Tail 80) -join "`n"
}

$dirtyBlock = if ($dirtyCount -eq 0) { '干净' } else { ($dirty -join "`n") }
$changedBlock = if ($changedFiles.Count -eq 0) { '（无变更）' } else { ($changedFiles -join "`n") }
$fence = '```'

$markdown = @"
# 自动代码审查基线报告

- 生成时间：$startedAt
- 仓库：$RepoRoot
- 当前分支：$branch
- HEAD：$headBefore → $head（$shortHead）
- origin/$BaseBranch：$originHead
- 工作区：$dirtyCount 个改动
- 拉取：$fetchNote
- 快进：$mergeNote
- 变更范围：$range
- 门禁：exit $verifyExit
- 生产依赖审计：$auditSummary

## 本次变更

${fence}text
$changedBlock
${fence}

## diff 统计

${fence}text
$diffStat
${fence}

## 门禁输出（末 80 行）

${fence}text
$verifyTail
${fence}

## 审查清单（需要人或 AI 继续判断）

- 正确性：边界条件、并发/异步、事务与幂等、时间/时区口径。
- 安全：认证与授权、注入、敏感信息、依赖与部署默认值。
- 可维护性：命名、分层依赖、重复、注释与文档一致性。
- 性能：明显低效查询、无上限分页、首屏包体。
- 实验指导书对照：需求/设计/原型/测试/总结五章、UML、Git 协作、AI 使用记录、积分核心测试。

$assignmentBlock

> 本报告只证明“基线命令执行结果”，不证明语义审查已完成；人工/AI 结论应追加在下方。
"@

Set-Content -LiteralPath $latestMarkdown -Value $markdown -Encoding utf8
Set-Content -LiteralPath (Join-Path $reportRoot "report-$stamp.md") -Value $markdown -Encoding utf8

$summary = [ordered]@{
    status       = $status
    startedAt    = $startedAt
    repo         = $RepoRoot
    branch       = $branch
    headBefore   = $headBefore
    head         = $head
    originHead   = $originHead
    dirtyCount   = $dirtyCount
    fetchNote    = $fetchNote
    mergeNote    = $mergeNote
    range        = $range
    changedFiles = @($changedFiles)
    verifyExit   = $verifyExit
    verifyLog    = $verifyLog
    audit        = $auditSummary
    report       = $latestMarkdown
    assignmentSource = $assignmentPath
    assignmentWarning = $assignmentWarning
    assignments  = @($assignmentRecords)
    unassignedAssignments = @($unassignedRecords | ForEach-Object { $_.Task })
}
$summary | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $latestJson -Encoding utf8

# 可选通知：设置 REVIEW_WEBHOOK_URL 后 POST 一份精简 JSON（不发送源码/日志）。
if ($env:REVIEW_WEBHOOK_URL) {
    $payload = [ordered]@{
        text      = "代码审查基线：$status；HEAD=$shortHead；门禁 exit=$verifyExit；$auditSummary"
        status    = $status
        head      = $head
        branch    = $branch
        verifyExit = $verifyExit
        audit     = $auditSummary
        report    = $latestMarkdown
    } | ConvertTo-Json -Depth 4
    try {
        Invoke-RestMethod -Uri $env:REVIEW_WEBHOOK_URL -Method Post -ContentType 'application/json' -Body $payload -TimeoutSec 10 | Out-Null
        Write-Host 'REVIEW_NOTIFY=webhook-ok'
    } catch {
        Write-Host "REVIEW_NOTIFY=webhook-failed: $($_.Exception.Message)"
    }
}

Write-Host ''
Write-Host "REVIEW_STATUS=$status"
Write-Host "REVIEW_HEAD=$shortHead"
Write-Host "REVIEW_VERIFY_EXIT=$verifyExit"
Write-Host "REVIEW_AUDIT=$auditSummary"
Write-Host "REVIEW_ASSIGNMENTS=$($assignmentRecords.Count)"
Write-Host "REVIEW_UNASSIGNED=$($unassignedRecords.Count)"
Write-Host "REVIEW_REPORT=$latestMarkdown"

if ($status -eq 'VERIFY_FAILED') { exit 1 }
if ($status -eq 'FETCH_FAILED' -or $status -eq 'PULL_FAILED') { exit 2 }
if ($status -eq 'SECURITY_CRITICAL') { exit 3 }
exit 0
