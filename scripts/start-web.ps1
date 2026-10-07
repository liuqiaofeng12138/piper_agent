# 一键启动：Runtime + Web Crawler Worker + Gateway（前端仍需 npm run dev）
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Config = Join-Path $Root "deploy\config\local.yaml"

Push-Location (Join-Path $Root "gateway")
try {
  go run ./cmd/piper-serve -f $Config
} finally {
  Pop-Location
}
