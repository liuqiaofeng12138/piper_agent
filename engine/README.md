# engine（Piper Go 采集内核）

本目录是 **piper_agent Monorepo** 的执行内核：Go 模块名仍为 `piper_go`，由 [runtime/](../runtime/) 通过 `replace piper_go => ../engine` 引用。Agent 默认不单独起 REST，而是 `piper-runtime` 内嵌 `pkg/agentruntime`；需要 Web UI / 原 HTTP API 时再编译运行 `pipergo.go`。

使用 [go-zero](https://go-zero.dev/) 完全重写 Java 版 Piper（`src/main/java/one/rewind`），与现有前端 `web/` 及 Docker 部署保持 **HTTP 路径、JSON 形态、WebSocket 协议、端口与行为** 对齐。

## 当前脚手架状态

| 路径 | 说明 |
|------|------|
| `pipergo.go` | REST 服务入口（goctl 生成，后续扩展 bootstrap） |
| `piper_go.api` | 占位 API，将拆分到 `api/*.api` 并用 goctl 生成 handler/logic/types |
| `internal/config` | `rest.RestConf`，需扩展为 Piper 全量配置 |
| `internal/handler` / `internal/logic` / `internal/svc` / `internal/types` | goctl 默认层，按域分子目录 |
| `etc/pipergo-api.yaml` | 运行配置（Java 默认 HTTP **81**，重构阶段可先用 8888，上线前对齐） |

## 分层原则（标准 go-zero + 领域包）

```
请求 → middleware → handler（薄） → logic（编排） → pkg/*（领域与基础设施）
                ↘ bootstrap 启动时初始化 distributor、ES/S3、定时任务
```

- **handler / logic**：仅 HTTP/WebSocket 接入与用例编排，不写复杂业务。
- **pkg/**：可测试的核心能力，对应 Java `one.rewind.nio.*`、`one.rewind.db.*` 等。
- **api/**：API 契约单一来源，与 `docs/API_INVENTORY.md` 一致。
- **docs/**：对照表、分期计划、验收清单。

## 目录总览

| 目录 | 职责 |
|------|------|
| [api/](api/README.md) | goctl `.api` 定义（按业务域拆分） |
| [docs/](docs/README.md) | API 清单、Java 包映射、实施阶段 |
| [etc/](etc/README.md) | YAML/环境配置 |
| [internal/bootstrap/](internal/bootstrap/README.md) | 启动流程（对齐 `WebAPI.main`） |
| [internal/middleware/](internal/middleware/README.md) | CORS、鉴权、Ready、响应头 |
| [internal/config/](internal/config/README.md) | 配置结构体 |
| [internal/svc/](internal/svc/README.md) | `ServiceContext` 依赖注入 |
| [internal/handler/](internal/handler/README.md) | HTTP handlers（按域分子目录） |
| [internal/logic/](internal/logic/README.md) | 业务逻辑（按域分子目录） |
| [internal/types/](internal/types/README.md) | goctl 生成的请求/响应类型 |
| [pkg/](pkg/README.md) | 领域与基础设施实现 |

## Java 主入口对照

| Java | Go |
|------|-----|
| `one.rewind.nio.web.WebAPI` | `pipergo.go` + `internal/bootstrap` |
| `one.rewind.nio.web.route.Routes` | `api/*.api` + `internal/handler` + `internal/middleware` |
| `one.rewind.nio.util.InitUtil` | `pkg/bootstrap` + `internal/bootstrap` |
| Spark WebSocket `/msg`, `/token_msg` | `pkg/websocket` + 独立 WS 服务或 go-zero 扩展 |

## 必读文档

1. [docs/API_INVENTORY.md](docs/API_INVENTORY.md) — 全部 REST 路由
2. [docs/JAVA_MAPPING.md](docs/JAVA_MAPPING.md) — Java 包 → Go 目录
3. [docs/IMPLEMENTATION_PHASES.md](docs/IMPLEMENTATION_PHASES.md) — 推荐实施顺序与验收

## 本地启动（Windows）

```powershell
cd engine
go build -o pipergo.exe .
.\pipergo.exe -f etc/pipergo-api.yaml
```

`go run .` 在 Windows 上会多一层 `go` 父进程；**Ctrl+C 有时只结束父进程，子进程仍占用 8888**。若出现 `bind: Only one usage of each socket address`，查占用并结束：

```powershell
Get-NetTCPConnection -LocalPort 8888 -State Listen | Select-Object OwningProcess
Stop-Process -Id <OwningProcess> -Force
```

服务收到 Ctrl+C 后会调用 `server.Stop()` 释放端口（见 `pipergo.go`）。

## 本地启动（Windows）

```powershell
cd engine
go build -o pipergo.exe .
.\pipergo.exe -f etc/pipergo-api.yaml
```

优先使用 **`go build` + 可执行文件**，少用 `go run .`：在 Windows 上 Ctrl+C 有时只结束 `go` 父进程，**编译出的 `piper_go` 子进程仍会占用 8888**，表现为“关掉了但端口仍被占用”。

释放 8888：

```powershell
Get-NetTCPConnection -LocalPort 8888 -ErrorAction SilentlyContinue |
  Select-Object -ExpandProperty OwningProcess -Unique |
  ForEach-Object { Stop-Process -Id $_ -Force }
```

## 开发流程约定

1. 在 `api/` 增加/修改 `.api` → `goctl api go` 生成 types/handler/logic 骨架。
2. 在对应 `internal/logic/<domain>` 调用 `pkg` 实现。
3. 行为以 Java 路由类 + 集成测试（`src/test/.../web/test/*RouteTest.java`）为验收基准。
4. 每个目录下的 `README.md` 说明该处应实现的代码类型及 Java 源文件索引。
