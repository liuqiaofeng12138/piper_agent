# engine（Piper 采集内核）

本目录是 **piper_agent** 的执行内核（Go 模块名 `piper_go`）。由 [runtime/](../runtime/) 通过 `replace piper_go => ../engine` 引用；`piper-runtime` 从 [deploy/config/local.yaml](../deploy/config/local.yaml) 的 **`engine:`** 段加载 ES/S3/meta/Chrome 等设置，并调用 `pkg/agentruntime` 完成采集。

原独立 REST 后端与 `etc/pipergo-api.yaml` 已移除；配置与 Agent 合并在 `deploy/config/local.yaml`。

## 目录

| 路径 | 说明 |
|------|------|
| [pkg/agentruntime/](pkg/agentruntime/) | 对外嵌入 API（Bootstrap / Validate / Run / GetTokenData） |
| [pkg/tpl/](pkg/tpl/) | 模版解析、HTTP/Chrome 构建与执行 |
| [pkg/distributor/](pkg/distributor/) | Token 队列与 Agent 调度 |
| [pkg/persistence/](pkg/persistence/) | ES/S3 写入与查询 |
| [pkg/chrome/](pkg/chrome/) | Chrome Agent 池 |
| [pkg/db/](pkg/db/) | meta SQLite、ES 客户端 |
| [internal/bootstrap/](internal/bootstrap/) | 启动编排 |
| [etc/db/](etc/db/) | 本地 meta SQLite 与 `agents_info.json`（路径由 `local.yaml` 中 `engine.H2.path` 指定） |

## 本地验证

```powershell
cd engine
go test ./...
```

Runtime 联调见仓库根 [README.md](../README.md) 与 [docs/PHASE_B.md](../docs/PHASE_B.md)。
