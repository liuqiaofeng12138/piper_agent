# Piper HTTP / WebSocket API 清单

来源：`src/main/java/one/rewind/nio/web/route/Routes.java` 及 `WebAPI.buildHttpApiServer`。

默认端口：**81**（Java Spark）。响应默认 `application/json` + gzip（`/metrics` 除外）。

## 全局中间件行为（需在 Go middleware 复现）

| 行为 | 说明 |
|------|------|
| CORS | `OPTIONS /*`；`Access-Control-*` 头 |
| Ready | 非 `InitRoutes` 且服务未 ready → **500** `Server not ready` |
| 鉴权 | `noAuth=false` 时，除 `NoAuthRoutes` 外需 token（`Authenticator`） |
| NoAuthRoutes | `/misc/info`, `/metrics`, `/msg`, `/auth/login`（正则匹配，大小写不敏感） |
| InitRoutes | `/misc/info`, `/misc/config`, `/misc/import`, `/misc/restart`（ready 检查豁免） |
| AdminRoutes | 路径含 `/admin`（角色校验，`Routes.hasRole`） |

## WebSocket

| 路径 | Java 类 | 说明 |
|------|---------|------|
| `GET /msg` | `MsgPublisher` | 广播通知、系统消息 |
| `GET /token_msg` | `TokenPublisher` | Token 执行过程推送 |

## Auth — `/auth`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| POST | `/auth/login` | `AuthRoute.login` |
| POST | `/auth/logout` | `AuthRoute.logout` |
| GET | `/auth/me` | `AuthRoute.me` |

## Misc — `/misc`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/misc/config` | `MiscRoute.getConfig` |
| PUT | `/misc/config` | `MiscRoute.updateConfig` |
| POST | `/misc/restart` | `MiscRoute.restart` |
| POST | `/misc/export` | `TaskRoute.exportPack` |
| POST | `/misc/import` | `TaskRoute.importPack` |
| POST | `/misc/run_token` | `TokenRoute.run` |
| GET | `/misc/exceptions` | `MiscRoute.exceptions` |
| GET | `/misc/info` | `MiscRoute.info`（节点信息，Prometheus 发现用） |
| GET | `/misc/global_vars` | `MiscRoute.get_global_vars` |

### Client 路由（`initClientRoutes`）

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/misc/data_migration/config` | `MiscRoute.data_migration_config_query` |
| PUT | `/misc/data_migration/config` | `MiscRoute.data_migration_config` |
| GET | `/misc/data_migration/logs` | `MiscRoute.data_migration_logs` |

## Data — `/data`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| POST | `/data/search` | `DataRoute.search` |
| POST | `/data/search_date_histogram` | `DataRoute.search_date_histogram` |

## Metrics — `/metrics`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/metrics` | Prometheus 文本，`Stats.getInstance()`，`Content-Type: text/plain; version=0.0.4` |

## Agents — `/agents`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/agents` | `AgentRoute.query` |
| POST | `/agents` | `AgentRoute.create` |
| GET | `/agents/:id` | `AgentRoute.get` |
| DELETE | `/agents/:id` | `AgentRoute.delete` |
| POST | `/agents/:id/accounts` | `AgentRoute.addDomainUsername` |
| DELETE | `/agents/:id/accounts` | `AgentRoute.removeDomainUsername` |
| POST | `/agents/:id/proxy` | `AgentRoute.setProxy` |
| POST | `/agents/:id/trigger` | `AgentRoute.trigger` |

## Proxies — `/proxies`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/proxies` | `ProxyRoute.query` |
| GET | `/proxies/:id` | `ProxyRoute.get` |
| POST | `/proxies` | `ProxyRoute.create` |
| PUT | `/proxies/:id` | `ProxyRoute.update` |
| DELETE | `/proxies/:id` | `ProxyRoute.delete` |
| POST | `/proxies/:id/init` | `ProxyRoute.init` |

## Accounts — `/accounts` 及扩展

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/accounts` | `AccountRoute.query` |
| GET | `/accounts/:id` | `AccountRoute.get` |
| POST | `/accounts` | `AccountRoute.create` |
| PUT | `/accounts/:id` | `AccountRoute.update` |
| DELETE | `/accounts/:id` | `AccountRoute.delete` |
| GET | `/accounts_filter` | `AccountRoute.filter` |
| POST | `/accounts_domains` | `AccountRoute.queryDomains` |
| POST | `/accounts_usernames` | `AccountRoute.queryUsernames` |

## Indices — `/indices`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/indices` | `IndexRoute.query` |
| GET | `/indices/:id` | `IndexRoute.get` |
| POST | `/indices` | `IndexRoute.create` |
| PUT | `/indices/:id` | `IndexRoute.update` |
| DELETE | `/indices/:id` | `IndexRoute.delete` |
| GET | `/indices/:id/ref` | `IndexRoute.ref` |
| GET | `/indices/:id/views` | `IndexRoute.views` |

## Vars lists — `/vars_lists`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/vars_lists` | `VarsListRoute.query` |
| GET | `/vars_lists/:id` | `VarsListRoute.get` |
| POST | `/vars_lists` | `VarsListRoute.create` |
| PUT | `/vars_lists/:id` | `VarsListRoute.update` |
| DELETE | `/vars_lists/:id` | `VarsListRoute.delete` |
| GET | `/vars_lists/:id/check` | `VarsListRoute.check` |
| GET | `/vars_lists/:id/1` | `VarsListRoute.first` |

## Funcs — `/funcs`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/funcs` | `FuncRoute.query` |
| GET | `/funcs/:id` | `FuncRoute.get` |
| POST | `/funcs` | `FuncRoute.create` |
| PUT | `/funcs/:id` | `FuncRoute.update` |
| DELETE | `/funcs/:id` | `FuncRoute.delete` |

## Templates — `/templates`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/templates` | `TemplateRoute.query` |
| GET | `/templates/mapper_types` | `TemplateRoute.mapper_types` |
| GET | `/templates/:id` | `TemplateRoute.get` |
| POST | `/templates` | `TemplateRoute.create` |
| PUT | `/templates/:id` | `TemplateRoute.update` |
| DELETE | `/templates/:id` | `TemplateRoute.delete` |
| GET | `/templates/:id/ref` | `TemplateRoute.ref` |
| GET | `/templates/:id/var_names` | `TemplateRoute.var_names` |
| POST | `/templates/:id/run` | `TemplateRoute.run` |

## Tasks — `/tasks`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/tasks` | `TaskRoute.query` |
| GET | `/tasks/:id` | `TaskRoute.get` |
| POST | `/tasks` | `TaskRoute.create` |
| PUT | `/tasks/:id` | `TaskRoute.update` |
| DELETE | `/tasks/:id` | `TaskRoute.delete` |
| POST | `/tasks/:id/copy` | `TaskRoute.copy` |
| POST | `/tasks/:id/run` | `TaskRoute.run` |
| POST | `/tasks/:id/stop` | `TaskRoute.stop` |

## Tokens — `/tokens`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/tokens` | `TokenRoute.query` |
| GET | `/tokens/:id` | `TokenRoute.get` |
| GET | `/tokens/:id/descendants` | `TokenRoute.descendants` |
| GET | `/tokens/:id/data` | `TokenRoute.data` |
| GET | `/tokens/:id/next` | `TokenRoute.next` |

## Logs — `/logs`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/logs` | `LogRoute.query` |

## Notifications — `/notifications`

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/notifications` | `NotificationRoute.query` |
| PUT | `/notifications/:id/read` | `NotificationRoute.read` |
| DELETE | `/notifications/:id` | `NotificationRoute.delete` |

## Nodes — `/nodes`（Client）

| 方法 | 路径 | Handler 参考 |
|------|------|----------------|
| GET | `/nodes` | `NodeRoute.list` |
| GET | `/nodes/:id` | `NodeRoute.get` |
| POST | `/nodes` | `NodeRoute.create` |
| DELETE | `/nodes/:id` | `NodeRoute.delete` |

## 后台定时任务（非 REST，须在 bootstrap 实现）

| 任务 | 来源 | 说明 |
|------|------|------|
| Prometheus 节点发现 | `WebAPI.buildHttpApiServer` | 每 30s 查询 `prometheusHost`，同步 `NodeInfo` |
| 本机 Proxy 信息刷新 | 同上 | 每 5s `node.initNodeProxyInfo` |
| Agents 快照写盘 | 同上 | 写 `agents` 文件 |
| FileTransporter | `FileTransporter.getInstance` | 数据迁移 ES/S3 → MySQL/FTP |
| Task Scheduler | `Scheduler` | 定时任务执行 |
| Proxy 校验 | `ProxyValidator` | 代理健康 |

## 分页查询约定

`Routes.QueryInfo`：`st`, `et`, `page`, `size` 查询参数（列表类接口通用）。
