# Phase B — 对接 piper_go（已实现）

## 能力

| gRPC | 实现 |
|------|------|
| `ValidateTemplate` | HTTP 模版 `BuildHTTPToken` 校验；可按 `template_id` 从 meta 加载 |
| `RunTemplate` | `distributor.Engine.RunTemplate`，`run_id` = `token_id` |
| `SubscribeRun` | 轮询 ES token 文档直至完成 |
| `ListTemplates` / `GetTemplate` | meta SQLite `templates` 表 |
| `GetTokenData` | 对齐 `TokenRoute.data`（`persistence.TokenData`） |
| `UpsertTemplate` | 写入 meta + cache |

## engine（piper_go）入口

- 目录：`engine/`（Go 模块名仍为 `piper_go`）
- Agent 嵌入包：`piper_go/pkg/agentruntime`（Bootstrap + Service API）
- Runtime 配置：`mock: false` + `piper_go_config: "../../engine/etc/pipergo-api.yaml"`（相对 `deploy/config/`）
- `relax_deps: true`：不强制 Docker 容器检查（与独立 API 进程一致时可开 `requireDeps`）
- `skip_storage_wait: false`：启动前等待 ES/S3 可达（与 Piper dev 栈一致）

## 启动

```powershell
# 需 ES 等依赖就绪（docker/piper_dev.yaml）
cd piper_agent\runtime
go build -o piper-runtime.exe ./cmd/piper-runtime
.\piper-runtime.exe -f ..\deploy\config\runtime.local.yaml
```

Mock 模式：配置中 `mock: true`，行为同 Phase A。

## REPL 新命令

- `list` — 列出模版库
- `validate <tpl_id>` — 按 meta 中模版 id 校验
- `data` — 拉取上次 run 的 token 聚合数据
