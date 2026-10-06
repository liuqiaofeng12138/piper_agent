# Phase A — 契约与骨架（已实现）

## 交付物

| 项 | 路径 |
|----|------|
| Protobuf 契约 | `proto/common/v1/*`, `proto/runtime/v1/*` |
| 生成 Go 代码 | `runtime/pkg/pb/**`（`scripts/gen_proto.ps1`） |
| 生成 Python 代码 | `agents/src/piper_agent/pb/**` |
| Mock Runtime（Go） | `runtime/internal/mock`, `runtime/cmd/piper-runtime` |
| gRPC Server | `runtime/internal/grpcserver` |
| Python gRPC 客户端 | `agents/src/piper_agent/clients/runtime_client.py` |
| 最小 Harness REPL | `agents/src/piper_agent/harness/repl.py` |
| RPC 错误约定 | `docs/RPC_ERRORS.md` |
| 冒烟测试 | `agents/tests/test_phase_a_smoke.py`, `runtime/internal/mock/runtime_test.go` |

## 本地运行

### 1. 启动 Mock Runtime

```powershell
cd piper_agent\runtime
go build -o piper-runtime.exe ./cmd/piper-runtime
.\piper-runtime.exe -listen :50051
# 或指定配置：.\piper-runtime.exe -f ..\deploy\config\local.yaml
```

### 2. Python REPL

```powershell
cd piper_agent\agents
pip install -e .
piper-agent repl --runtime localhost:50051
```

REPL 命令：

- `validate demo-tpl` — 调用 `ValidateTemplate`
- `run demo-tpl` — 调用 `RunTemplate`（须先 validate）
- `watch` — `SubscribeRun` 订阅进度

### 3. 冒烟脚本（无需 REPL）

```powershell
$env:PIPER_RUNTIME_ADDR="localhost:50051"
python piper_agent\agents\tests\test_phase_a_smoke.py
```

## Mock 行为说明

- **ValidateTemplate**：要求 `name` 或 `id`，且 `json_payload` 非空；成功则记入「已校验」集合。
- **RunTemplate**：`template_id` 必须先 Validate；返回 `run_id`，后台约 0.7s 模拟 Running → Succeeded。
- **SubscribeRun**：推送 mock 过程消息。
- **CancelRun**：取消进行中的 mock run。
- **UpsertTemplate**：分配 `tpl-{uuid}`（Phase A 未与 Validate 强制绑定）。

Phase B 将替换 mock 为真实 `piper_go` 执行。
