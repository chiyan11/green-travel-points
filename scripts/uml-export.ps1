<#
.SYNOPSIS
    批量渲染 docs\uml\*.puml → docs\uml\out\（PNG / SVG）。

.DESCRIPTION
    这是在本机（Windows + Java 25，无 Graphviz）实测跑通的参数组合。
    三个坑已在脚本里规避，手工敲命令时也请照抄：

      1) PowerShell 会把裸参数 -Dfile.encoding=UTF-8 拆坏
         （报 "找不到或无法加载主类 .encoding=UTF-8"）
         → 统一通过 $env:JAVA_TOOL_OPTIONS 传递 JVM 参数。

      2) 系统临时目录不可写时，imageio 会抛 "Can't create cache file!"，
         导出直接失败；PlantUML 顺带还会在 %TEMP% 里解压 Graphviz。
         → 用 -Djava.io.tmpdir 指到工作区内的 .tools\tmp。

      3) PlantUML 默认画布上限 4096px，宽图（如 19 个用例的总览图）
         会被**静默裁切**，右侧内容凭空消失。
         → -DPLANTUML_LIMIT_SIZE=20000。

    没有 Graphviz 时自动加 -Playout=smetana（纯 Java 布局，样式略方正）。

.EXAMPLE
    .\scripts\uml-export.ps1                 # 导出 PNG
    .\scripts\uml-export.ps1 -Format svg     # 导出 SVG（Word 里更清晰）
    .\scripts\uml-export.ps1 -CheckOnly      # 只做语法校验，不出图（秒级）
#>
[CmdletBinding()]
param(
    [ValidateSet('png', 'svg')]
    [string]$Format = 'png',

    # 只校验语法，不生成图片：提交前跑一次，比逐张看图快得多
    [switch]$CheckOnly,

    # 强制使用内置 smetana 布局（即使装了 Graphviz 也不用）
    [switch]$Smetana,

    # 输出子目录，相对 docs\uml\（PlantUML 的 -o 以源文件目录为基准）
    [string]$Out = 'out'
)

$ErrorActionPreference = 'Stop'

$root  = Split-Path -Parent $PSScriptRoot
$src   = Join-Path $root 'docs\uml'
$tools = Join-Path $src  '.tools'
$tmp   = Join-Path $tools 'tmp'
$jar   = Join-Path $tools 'plantuml.jar'

if (-not (Test-Path $src)) { throw "找不到图目录：$src" }

# ---------- 1. 准备 jar ----------
if (-not (Test-Path $jar)) {
    Write-Host "未找到 plantuml.jar，正在下载到 $tools ..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Force $tools | Out-Null
    $url = 'https://repo1.maven.org/maven2/net/sourceforge/plantuml/plantuml/1.2024.8/plantuml-1.2024.8.jar'
    Invoke-WebRequest -Uri $url -OutFile $jar -TimeoutSec 120
}

New-Item -ItemType Directory -Force $tmp | Out-Null

# ---------- 2. 设置 JVM 参数（见 .DESCRIPTION 的三条坑） ----------
$oldJavaOpts = $env:JAVA_TOOL_OPTIONS
$env:JAVA_TOOL_OPTIONS = (@(
    '-Dfile.encoding=UTF-8',
    "-Djava.io.tmpdir=$($tmp -replace '\\', '/')",
    '-DPLANTUML_LIMIT_SIZE=20000'
) -join ' ')

try {
    $files = @(Get-ChildItem $src -Filter *.puml | Sort-Object Name)
    if ($files.Count -eq 0) { throw "在 $src 下没有找到任何 .puml 文件" }

    $hasDot = [bool](Get-Command dot -ErrorAction SilentlyContinue)
    $layout = @()
    if ($Smetana -or -not $hasDot) {
        $layout = @('-Playout=smetana')
        Write-Host "布局引擎：smetana（未检测到 Graphviz，或已强制指定）" -ForegroundColor DarkGray
    } else {
        Write-Host "布局引擎：Graphviz (dot)" -ForegroundColor DarkGray
    }

    if ($CheckOnly) {
        # ---------- 3a. 语法校验 ----------
        Write-Host "语法校验：$($files.Count) 个文件 ..." -ForegroundColor Cyan
        $bad = @()
        foreach ($f in $files) {
            $null = & java -jar $jar -checkonly $f.FullName 2>&1
            if ($LASTEXITCODE -ne 0) { $bad += $f }

            # -checkonly 不给行号，失败时用 -syntax 取精确位置
            if ($LASTEXITCODE -ne 0 -and $f -eq $bad[-1]) {
                Write-Host "  ❌ $($f.Name)" -ForegroundColor Red
                $detail = (Get-Content $f.FullName -Raw -Encoding UTF8) |
                          & java -jar $jar -syntax 2>&1 | Out-String
                $detail -split "`n" |
                    Where-Object { $_.Trim() -ne '' -and $_ -notmatch 'Picked up JAVA_TOOL_OPTIONS' } |
                    ForEach-Object { Write-Host "     $_" -ForegroundColor DarkYellow }
            } else {
                Write-Host "  ✅ $($f.Name)" -ForegroundColor DarkGreen
            }
        }
        if ($bad.Count -gt 0) { throw "$($bad.Count) 个文件语法有误，见上" }
        Write-Host "全部通过。" -ForegroundColor Green
        return
    }

    # ---------- 3b. 渲染 ----------
    $dest = Join-Path $src $Out
    New-Item -ItemType Directory -Force $dest | Out-Null

    foreach ($f in $files) {
        Write-Host "渲染 $($f.Name) ..." -ForegroundColor Cyan
        $null = & java -jar $jar "-t$Format" -charset UTF-8 @layout -o $Out $f.FullName 2>&1
        if ($LASTEXITCODE -ne 0) { throw "渲染失败：$($f.Name)（先用 -CheckOnly 定位语法错误）" }
    }

    $made = @(Get-ChildItem $dest -Filter "*.$Format")
    Write-Host "完成：$($made.Count) 张图 → $dest" -ForegroundColor Green
}
finally {
    $env:JAVA_TOOL_OPTIONS = $oldJavaOpts
}
