# API 定义（goctl）

存放 `*.api` 文件，作为 **HTTP 契约的唯一来源**。通过 goctl 生成 `internal/types`、`internal/handler`、`internal/logic` 骨架。

## 推荐拆分文件

| 文件 | 对应 Java | 路由前缀 |
|------|-----------|----------|
| `base.api` | 公共 type、分页、统一响应 | — |
| `auth.api` | `AuthRoute` | `/auth` |
| `misc.api` | `MiscRoute`, 部分 `TaskRoute`/`TokenRoute` | `/misc` |
| `data.api` | `DataRoute` | `/data` |
| `metrics.api` | `Routes` metrics 块 | `/metrics` |
| `agent.api` | `AgentRoute` | `/agents` |
| `proxy.api` | `ProxyRoute` | `/proxies` |
| `account.api` | `AccountRoute` | `/accounts`, `/accounts_*` |
| `index.api` | `IndexRoute` | `/indices` |
| `varslist.api` | `VarsListRoute` | `/vars_lists` |
| `func.api` | `FuncRoute` | `/funcs` |
| `template.api` | `TemplateRoute` | `/templates` |
| `task.api` | `TaskRoute` | `/tasks` |
| `token.api` | `TokenRoute` | `/tokens` |
| `log.api` | `LogRoute` | `/logs` |
| `notification.api` | `NotificationRoute` | `/notifications` |
| `node.api` | `NodeRoute` | `/nodes` |

## 工作流

1. 编辑 `api/<domain>.api`（路径、method、request/response 与 Java 一致）。
2. 在 `piper_go` 根目录执行 goctl 合并生成（具体命令在首次编写 `.api` 时固定到根 README）。
3. 生成代码落入 `internal/handler/<domain>`、`internal/logic/<domain>`（可通过 goctl `--style` 或生成后移动对齐本仓库结构）。

## 注意事项

- 查询参数 `st`, `et`, `page`, `size` 在 `base.api` 定义为可复用类型。
- `/templates/mapper_types` 等 **静态路径** 须在 `.api` 中排在 `/:id` 之前，避免路由冲突。
- WebSocket **不** 用 REST `.api` 定义，见 `pkg/websocket/README.md`。

## 参考

- 路由清单：[docs/API_INVENTORY.md](../docs/API_INVENTORY.md)
- Java 注册：[Routes.java](../../src/main/java/one/rewind/nio/web/route/Routes.java)
