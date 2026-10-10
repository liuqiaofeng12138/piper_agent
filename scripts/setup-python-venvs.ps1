# 为每个 Python 子 Agent 创建独立 .venv 并安装依赖（目录需含 pyproject.toml）
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

# 可选 pip extras：按项目目录名配置；未列出的项目仅 pip install -e .
$ExtrasByDir = @{
    "web_crawler_agent" = "dev,llm"
    "general_agent"     = "llm"
    "rag_agent"         = "dev"
    "paper_agent"       = "dev"
}

function Get-PythonVersionTag([string]$exe) {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $out = & $exe -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')" 2>$null
    $ErrorActionPreference = $prev
    if ($LASTEXITCODE -ne 0) { return "" }
    return "$out".Trim()
}

function Find-Python {
    # Windows 优先 3.11–3.13（grpc/protobuf 在 3.14 上易出现残缺安装）
    if (Get-Command py -ErrorAction SilentlyContinue) {
        foreach ($tag in @("-3.11", "-3.12", "-3.13")) {
            $prev = $ErrorActionPreference
            $ErrorActionPreference = "SilentlyContinue"
            $exe = (& py $tag -c "import sys; print(sys.executable)" 2>$null | Select-Object -Last 1)
            $ErrorActionPreference = $prev
            if ($LASTEXITCODE -eq 0 -and $exe) {
                $path = "$exe".Trim()
                $ver = Get-PythonVersionTag $path
                Write-Host "使用 Python $ver ($path)"
                return $path
            }
        }
    }
    foreach ($name in @("python", "python3")) {
        if (-not (Get-Command $name -ErrorAction SilentlyContinue)) { continue }
        $path = (Get-Command $name).Source
        $ver = Get-PythonVersionTag $path
        if ($ver -match '^3\.(\d+)$' -and [int]$Matches[1] -ge 14) {
            Write-Warning "默认 Python $ver 可能不兼容 grpc/protobuf，建议安装 3.11 并执行: py -3.11 -m venv .venv"
        }
        Write-Host "使用 Python $ver ($path)"
        return $path
    }
    throw "未找到 python，请先安装 Python 3.11+（Windows 推荐从 python.org 安装并勾选 py launcher）"
}

function Test-VenvWorkerDeps([string]$venvPy) {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    & $venvPy -c "import google.protobuf; import grpc" 1>$null 2>$null
    $ok = $LASTEXITCODE -eq 0
    $ErrorActionPreference = $prev
    return $ok
}

function Get-VenvPython([string]$venvDir) {
    $win = Join-Path $venvDir "Scripts\python.exe"
    if (Test-Path $win) { return $win }
    $unix = Join-Path $venvDir "bin\python"
    if (Test-Path $unix) { return $unix }
    return $null
}

function Invoke-Quiet([string]$exe, [string[]]$args) {
    # 避免子进程输出进入 PowerShell 管道；忽略 stderr 横幅（如 python -m venv）
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    & $exe @args 1>$null 2>$null
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    if ($code -ne 0) {
        throw "命令失败: $exe $($args -join ' ') (exit $code)"
    }
}

function Test-VenvHasPip([string]$venvPy) {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    & $venvPy -c "import pip" 1>$null 2>$null
    $ok = $LASTEXITCODE -eq 0
    $ErrorActionPreference = $prev
    return $ok
}

function Remove-VenvDir([string]$venvDir) {
    if (-not (Test-Path $venvDir)) { return }
    try {
        Remove-Item -Recurse -Force $venvDir -ErrorAction Stop
    } catch {
        throw @"
无法删除 $venvDir（文件可能被 piper-serve 或其它进程占用）。
请先停止正在运行的 piper-serve / Python Worker，再重新执行本脚本。
$($_.Exception.Message)
"@
    }
}

function Ensure-Venv([string]$basePy, [string]$venvDir) {
    $baseVer = Get-PythonVersionTag $basePy
    $venvPy = Get-VenvPython $venvDir
    if ($venvPy -and $baseVer) {
        $venvVer = Get-PythonVersionTag $venvPy
        if ($venvVer -and $venvVer -ne $baseVer) {
            Write-Host "  .venv 为 Python $venvVer，目标为 $baseVer，正在重建..."
            Remove-VenvDir $venvDir
            $venvPy = $null
        }
    }
    if ($venvPy -and -not (Test-VenvWorkerDeps $venvPy)) {
        Write-Host "  .venv 缺少 grpc/protobuf（常见于 Python 3.14 残缺安装），正在重建..."
        Remove-VenvDir $venvDir
        $venvPy = $null
    }
    if ($venvPy -and -not (Test-VenvHasPip $venvPy)) {
        Write-Host "  .venv 中缺少 pip，尝试 ensurepip..."
        Invoke-Quiet $venvPy @("-m", "ensurepip", "--upgrade")
    }
    if ($venvPy -and (Test-VenvHasPip $venvPy)) {
        return $venvPy
    }

    if ($venvPy) {
        Write-Host "  ensurepip 未成功，正在重建 .venv ..."
        Remove-VenvDir $venvDir
        $venvPy = $null
    }
    if (-not $venvPy) {
        if (Test-Path $venvDir) {
            Write-Host "  检测到不完整的 .venv，正在重建..."
            Remove-VenvDir $venvDir
        }
        Invoke-Quiet $basePy @("-m", "venv", $venvDir)
        $venvPy = Get-VenvPython $venvDir
        if (-not $venvPy -and (Test-Path $venvDir)) {
            Remove-VenvDir $venvDir
            Invoke-Quiet $basePy @("-m", "venv", "--clear", $venvDir)
            $venvPy = Get-VenvPython $venvDir
        }
    }
    if (-not $venvPy) {
        throw "创建虚拟环境失败: $venvDir"
    }
    if (-not (Test-VenvHasPip $venvPy)) {
        Invoke-Quiet $venvPy @("-m", "ensurepip", "--upgrade")
    }
    if (-not (Test-VenvHasPip $venvPy)) {
        throw "虚拟环境中无法安装 pip: $venvDir"
    }
    return $venvPy
}

$py = Find-Python
$projects = Get-ChildItem -Path $Root -Directory | Where-Object {
    Test-Path (Join-Path $_.FullName "pyproject.toml")
}

if (-not $projects) {
    Write-Host "未在 $Root 下发现含 pyproject.toml 的 Python 项目"
    exit 1
}

foreach ($proj in $projects) {
    $name = $proj.Name
    $venv = Join-Path $proj.FullName ".venv"
    Write-Host "==> $name"
    [string]$venvPy = Ensure-Venv $py $venv
    $extras = $ExtrasByDir[$name]
    Push-Location $proj.FullName
    & $venvPy -m pip install -U pip wheel
    if ($LASTEXITCODE -ne 0) { throw "pip 升级失败: $name" }
    if ($extras) {
        & $venvPy -m pip install -e ".[$extras]"
    } else {
        & $venvPy -m pip install -e "."
    }
    if ($LASTEXITCODE -ne 0) { throw "依赖安装失败: $name" }
    if (-not (Test-VenvWorkerDeps $venvPy)) {
        throw "依赖自检失败（google.protobuf / grpc）: $name，请删除 $venv 后重试本脚本"
    }
    Pop-Location
}

Write-Host "完成。各 Agent 虚拟环境位于 <项目>/.venv，piper-serve 将严格按 Agent 使用对应 .venv。"
