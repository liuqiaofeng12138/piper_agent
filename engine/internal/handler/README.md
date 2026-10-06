# internal/handler

HTTP **接入层**（go-zero handler）：解析请求、调用 logic、写响应。

## 组织方式

按业务域分子目录，与 `api/*.api` 及 [docs/API_INVENTORY.md](../../docs/API_INVENTORY.md) 一一对应：

| 子目录 | Java Route |
|--------|------------|
| [auth](auth/README.md) | `AuthRoute` |
| [agent](agent/README.md) | `AgentRoute` |
| [proxy](proxy/README.md) | `ProxyRoute` |
| [account](account/README.md) | `AccountRoute` |
| [index](index/README.md) | `IndexRoute` |
| [varslist](varslist/README.md) | `VarsListRoute` |
| [func](func/README.md) | `FuncRoute` |
| [template](template/README.md) | `TemplateRoute` |
| [task](task/README.md) | `TaskRoute` |
| [token](token/README.md) | `TokenRoute` |
| [log](log/README.md) | `LogRoute` |
| [notification](notification/README.md) | `NotificationRoute` |
| [data](data/README.md) | `DataRoute` |
| [misc](misc/README.md) | `MiscRoute` + 部分 misc 路径 |
| [node](node/README.md) | `NodeRoute` |
| [metrics](metrics/README.md) | `/metrics` |

## 约定

- Handler 保持 **薄**：参数校验轻量，核心在 `internal/logic/<same>`。
- 根目录下 goctl 生成的 `pipergohandler.go`、`routes.go` 在拆分域 API 后迁移或替换为分域注册。
- 图片等特殊响应（`Routes.buildImageResponse`）在对应 handler 单独处理 Content-Type。

## 生成

由 goctl 从 `api/` 生成；生成后按子目录归档。
