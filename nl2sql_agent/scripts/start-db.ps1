$ErrorActionPreference = "Stop"
$Dir = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $Dir "docker")
docker compose up -d
docker compose ps
Pop-Location
