# engine（Piper 采集内核）

本目录是 **piper_agent** 的执行内核（Go 模块名 `piper_go`）。由 [runtime/](../runtime/) 通过 `replace piper_go => ../engine` 引用；`piper-runtime` 启动时加载 [etc/pipergo-api.yaml](etc/pipergo-api.yaml)，调用 `pkg/agentruntime` 完成模版校验、Token 调度、Chrome/HTTP 采集与 ES/S3 持久化。

原独立 REST 后端（`pipergo.go`、`internal/handler` 等）已移除，Agent 场景只保留库形态。

## 目录

| 路径 | 说明 |
|------|------|
| [pkg/agentruntime/](pkg/agentruntime/) | 对外嵌入 API（Bootstrap / Validate / Run / GetTokenData） |
| [pkg/tpl/](pkg/tpl/) | 模版解析、HTTP/Chrome 构建与执行 |
| [pkg/distributor/](pkg/distributor/) | Token 队列与 Agent 调度 |
| [pkg/persistence/](pkg/persistence/) | ES/S3 写入与查询 |
| [pkg/chrome/](pkg/chrome/) | Chrome Agent 池 |
| [pkg/db/](pkg/db/) | meta SQLite、ES 客户端 |
| [internal/bootstrap/](internal/bootstrap/) | 与 Runtime 共用的启动编排 |
| [internal/config/](internal/config/) | YAML 配置结构 |
| [internal/svc/](internal/svc/) | 运行时依赖容器 |
| [etc/pipergo-api.yaml](etc/pipergo-api.yaml) | ES/S3/H2/Chrome 等依赖配置 |

## 本地验证

```powershell
cd engine
go test ./...
```

Runtime 联调见仓库根 [README.md](../README.md) 与 [docs/PHASE_B.md](../docs/PHASE_B.md)。
