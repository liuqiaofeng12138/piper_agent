# Generate Go + Python stubs from piper_agent/proto
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$ProtoRoot = Join-Path $Root "proto"
$RuntimeOut = Join-Path $Root "runtime"
$AgentsPB = Join-Path $Root "agents\src\piper_agent\pb"

New-Item -ItemType Directory -Force -Path (Join-Path $RuntimeOut "pkg\pb") | Out-Null
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
  --python_out=$AgentsPB `
  --grpc_python_out=$AgentsPB `
  $protoFiles

Write-Host "Go stubs -> $RuntimeOut\pkg\pb"
Write-Host "Python stubs -> $AgentsPB"
Write-Host "Note: fix Python imports to piper_agent.pb.* in *_pb2*.py if regenerated"
