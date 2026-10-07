# Generate Go + Python stubs from piper_agent/proto
# Python 输出：web_crawler_agent（全部 proto，采集链路）+ general_agent（仅 agent/v1，通用 Worker 契约）
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$ProtoRoot = Join-Path $Root "proto"
$RuntimeOut = Join-Path $Root "runtime"
$GatewayOut = Join-Path $Root "gateway"
$ClawPB = Join-Path $Root "web_crawler_agent\src\web_crawler_agent\pb"
$AgentsPB = Join-Path $Root "general_agent\src\piper_agent\pb"

New-Item -ItemType Directory -Force -Path (Join-Path $RuntimeOut "pkg\pb") | Out-Null
New-Item -ItemType Directory -Force -Path $ClawPB | Out-Null
New-Item -ItemType Directory -Force -Path $AgentsPB | Out-Null

$protos = @(
    "common/v1/types.proto",
    "common/v1/errors.proto",
    "runtime/v1/template.proto",
    "runtime/v1/execute.proto",
    "runtime/v1/meta.proto",
    "runtime/v1/data.proto"
)

$protoFiles = $protos | ForEach-Object { Join-Path $ProtoRoot $_ }

protoc `
  -I $ProtoRoot `
  --go_out=$RuntimeOut --go_opt=module=piper_agent/runtime `
  --go-grpc_out=$RuntimeOut --go-grpc_opt=module=piper_agent/runtime `
  $protoFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$ClawPB `
  --grpc_python_out=$ClawPB `
  $protoFiles

$agentProtos = @("agent/v1/execute.proto")
$agentFiles = $agentProtos | ForEach-Object { Join-Path $ProtoRoot $_ }

protoc `
  -I $ProtoRoot `
  --go_out=$GatewayOut --go_opt=module=piper_agent/gateway `
  --go-grpc_out=$GatewayOut --go-grpc_opt=module=piper_agent/gateway `
  $agentFiles

# agent/v1 契约同时供 web_crawler_agent 与 general_agent 两个 Python 项目使用
python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$ClawPB `
  --grpc_python_out=$ClawPB `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$AgentsPB `
  --grpc_python_out=$AgentsPB `
  $agentFiles

python (Join-Path $PSScriptRoot "fix_pb_imports.py")

Write-Host "Go stubs -> $RuntimeOut\pkg\pb"
Write-Host "Go agent stubs -> $GatewayOut\pkg\pb"
Write-Host "Python stubs -> $ClawPB, $AgentsPB (imports fixed via fix_pb_imports.py)"
