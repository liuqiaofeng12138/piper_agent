# Generate Go + Python stubs from piper_agent/proto
# Python agent/v1 输出到各 Python 子 Agent 项目（采集已迁至 gateway 内 Go Worker，不再生成 web_crawler_agent stub）
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$ProtoRoot = Join-Path $Root "proto"
$RuntimeOut = Join-Path $Root "runtime"
$GatewayOut = Join-Path $Root "gateway"
$AgentsPB = Join-Path $Root "general_agent\src\piper_agent\pb"
$RagPB = Join-Path $Root "rag_agent\src\rag_agent\pb"
$PaperPB = Join-Path $Root "paper_agent\src\paper_agent\pb"
$Word2PPTPB = Join-Path $Root "word2ppt_agent\src\word2ppt_agent\pb"
$NL2SQLPB = Join-Path $Root "nl2sql_agent\src\nl2sql_agent\pb"

New-Item -ItemType Directory -Force -Path (Join-Path $RuntimeOut "pkg\pb") | Out-Null
New-Item -ItemType Directory -Force -Path $AgentsPB | Out-Null
New-Item -ItemType Directory -Force -Path $RagPB | Out-Null
New-Item -ItemType Directory -Force -Path $PaperPB | Out-Null
New-Item -ItemType Directory -Force -Path $Word2PPTPB | Out-Null
New-Item -ItemType Directory -Force -Path $NL2SQLPB | Out-Null

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

$agentProtos = @("agent/v1/execute.proto")
$agentFiles = $agentProtos | ForEach-Object { Join-Path $ProtoRoot $_ }

protoc `
  -I $ProtoRoot `
  --go_out=$GatewayOut --go_opt=module=piper_agent/gateway `
  --go-grpc_out=$GatewayOut --go-grpc_opt=module=piper_agent/gateway `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$AgentsPB `
  --grpc_python_out=$AgentsPB `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$RagPB `
  --grpc_python_out=$RagPB `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$PaperPB `
  --grpc_python_out=$PaperPB `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$Word2PPTPB `
  --grpc_python_out=$Word2PPTPB `
  $agentFiles

python -m grpc_tools.protoc `
  -I $ProtoRoot `
  --python_out=$NL2SQLPB `
  --grpc_python_out=$NL2SQLPB `
  $agentFiles

python (Join-Path $PSScriptRoot "fix_pb_imports.py")

Write-Host "Go stubs -> $RuntimeOut\pkg\pb"
Write-Host "Go agent stubs -> $GatewayOut\pkg\pb"
Write-Host "Python agent/v1 stubs -> general_agent, rag_agent, ... (fix_pb_imports.py)"
