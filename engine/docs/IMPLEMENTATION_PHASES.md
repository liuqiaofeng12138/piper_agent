# 实施阶段建议

目标：**功能与 Java Piper 对齐**，可按阶段交付、每阶段可独立验收。

## Phase 0 — 工程基座（当前）

- [x] go-zero 脚手架 `piper_go`
- [x] 目录规划与各文件夹 README
- [x] 扩展 `internal/config`（对齐 `WebAPI.conf`）
- [x] `internal/middleware`：CORS、Ready、鉴权开关（`noAuth` + Bearer）
- [x] `pkg/json`：统一 API 响应 envelope（对齐 `JsonTransformer` / `Msg`）
- [x] `internal/bootstrap` + `pkg/bootstrap`：存储等待（可 `skipStorageWait`）、`ready`、节点快照
- [x] Phase 0 路由：`GET /misc/info`、`GET /metrics`
- [x] 端口：`etc/pipergo-api.yaml` 默认 8888，生产可对齐 Java **81**（见 `etc/README.md`）

## Phase 1 — 存储与元数据 CRUD

依赖：`pkg/db/meta`（SQLite 元数据，对齐 Java H2 表名与 JSON 形态）

- [x] `/auth/login|logout|me`
- [x] `/indices/*`（含 `ref`, `views`；get 支持 id 或 name）
- [x] `/funcs/*`, `/vars_lists/*`（含 `check`, `/:id/1`）
- [x] `/accounts/*`, `/accounts_filter`, `/accounts_domains`, `/accounts_usernames`
- [x] `/proxies/*`（`init` 为占位，PPP setup 后续阶段）
- [x] `/templates/*`（含 `mapper_types`, `var_names`, `ref`；不含 `run`）
- [x] `/tasks/*` CRUD + `copy`（不含 `run|stop|export|import`）
- [x] `/misc/config`, `/misc/info`, `/misc/global_vars`, `/misc/exceptions`

说明：实体以 JSON payload 持久化，ID 规则与 Java `setId()` 一致；Index ES 建索引、VarsList S3/ES 拉取、Template `updateVarMappers` 等在 Phase 2+ 深化。

验收：对应 `*RouteTest.java` + 前端配置页可读写。

## Phase 2 — 调度内核（Distributor）

- [x] `pkg/distributor` 队列 + HTTP Token 执行 + ES 持久化（index `token`）
- [x] `pkg/db/es` 客户端（search/get/index）
- [x] `pkg/tpl` HTTP `BuildHTTPToken`（`{{var}}` 替换、Token id/fingerprint）
- [x] `/templates/:id/run`, `/misc/run_token`, `/tasks/:id/run|stop`
- [x] `/tokens` query/get/next/data/descendants
- [x] `/logs`（ES index `log`）
- [x] WebSocket `GET /msg`, `GET /token_msg`（`Sec-WebSocket-Protocol` 鉴权）
- [x] `/metrics` 增加 `piper_token_queue`

说明：Chrome 模板 run 返回与 Java 相同错误；`tokens/data|descendants` 尚未解析 Log.Proc 树；Task 未实现 cron/vars_list ES 拉取/full Mapper 链（Phase 3）。

验收：`TemplateRouteTest`, `TaskRouteTest`, `TokenRouteTest`。

## Phase 3 — 模板引擎完整度

### Phase 3a — HTTP（已完成）

- [x] HTTP 请求后执行 `procedures`（`Template.run` 等价，跳过 `Interceptor` 主循环）
- [x] `Mapper`：Regex / Line / JSONPath / Selector（goquery）、Index / Template / Source 产出
- [x] `If`（CheckSource / CheckVars）、`For`、`IdleAction`、`FuncCallAction`、`LoadUrlAction`（HTTP GET）
- [x] `pkg/piperfunc`：对齐 Java `Func.call`（meta `functions` 表）
- [x] `pkg/distributor`：执行后跑过程、子 Token 入队、Req/Proc 日志写入 token
- [x] `/tokens/:id/data|descendants` 聚合 Log.Proc 树（docs / sources / 子 token 递归）
- [x] 单元测试：`pkg/tpl/*_test.go`（正则 `(?<T>)`、evalRule、Mapper Index）

说明：未实现 gzip 响应体、YouTube/Bilibili Mapper、ContentCleaner 图片 Source、完整 JS `Evaluator` 边界；Task vars_list ES 拉取仍为 Phase 2 简化。

### Phase 3b — Chrome（已完成基线）

- [x] `BuildChromeToken` / `RunTemplate` 走 `ChromeDistributor`（无 Agent 时错误与 Java 一致）
- [x] `pkg/chrome`：chromedp 常驻 Agent 池、`Navigate` + `window.stop`、页面过程执行
- [x] `Interceptor`：CDP `Network` 捕获响应，URL regex 匹配后 Source / Mapper（对齐 `ProxyResponseFilter`）
- [x] Chrome Action：`ScrollAction`、`ClickAction`、`SetValueAction`、`RedirectAction`、`ExecAction`(script)
- [x] `If.CheckElement`、`For` DOM 规模退出条件
- [x] 配置 `Chrome.enabled|headless|agentCount|binaryPath`（默认 `enabled: false`）

说明：未实现 Java Docker `ChromeContainer`/MITM BMP、`LoginAction` 账号绑定、xdotool 滚轮、截图 `SHOOT_SCREEN`、Agent CRUD 路由；启用 Chrome 需本机 Chromium/Chrome 与 `Chrome.enabled: true`。

## Phase 4 — 数据面（已完成基线）

- [x] `pkg/persistence`：`DocForES`（对齐 `DocAdapter`）、`SourceForES`、`Persister`（ES 文档 + S3 `source` 桶后写 ES）
- [x] Token 成功后 `Persister.FromToken`（HTTP / Chrome 引擎）
- [x] `POST /data/search`, `POST /data/search_date_histogram`（对齐 `DataRoute` 查询参数）
- [x] `/tokens/:id/data`：docs 按 **Index.name** 分组（对齐 Java `TokenRoute.data`）
- [x] `pkg/log`：`Appender` 批量写入 ES index `log`
- [x] `pkg/storage`：MinIO/S3 客户端（`endpointUrl` + 配置密钥）

说明：Influx 时序索引、Kryo Token 快照 S3、raw/Essay 等 raw models 未做；字段 cast 按 Index 元数据简化。

验收：`DataRouteTest` 风格查询；Mapper 产出 doc 可在 ES 对应 index 检索。

## Phase 5 — 集群与运维

- [x] `/nodes/*`（list/get/create/delete）；本地节点写入 `meta.nodes`；`StartPrometheusSync`（`WebAPI.prometheusHost`，约 30s）
- [x] `pkg/cluster/FileTransporter` + `/misc/data_migration/config|logs`
- [x] `POST /misc/export|import`（`cluster.Pack`）
- [x] `POST /misc/restart` → `db/agents_info.json`（与 H2 同目录）
- [x] `/agents/*`（内存 registry + Chrome `AddAgent`/`RemoveAgent`；Http 仅 registry）
- [x] `/metrics`（Phase 1，`distributor.Stats`）

说明：FileTransporter 无 FTP/MySQL 后台同步；Agent 重启仅恢复 registry 快照，Chrome 仍由 `Chrome.enabled` 启动。

验收：`NodeRouteTest`, `ProxyRouteTest`, `AgentRouteTest`, `MiscRouteTest`（Go 侧以手工/API 对照 Java）。

## Phase 6 — 边缘能力

- [x] `pkg/network/ssh`：`SshHost` 等价（密码/私钥、`Exec`、本地端口转发）
- [x] `pkg/proxy/impl`：`ProxyPPPD.Setup/Close`；`POST /proxies/:id/init` 对齐 Java setup+close
- [x] `pkg/docker`：`DockerHost`/`ChromeContainer` 占位（Chrome 仍用 chromedp 本地）
- [x] `pkg/agent/android`：Agent 占位（无 ADB 执行）
- [x] `pkg/txt`：`StringUtil`/`URLUtil`/`ContentCleaner`/`distance.Levenshtein`
- [x] `pkg/simulator/mouse`：轨迹缩放（`ScalePath`）
- [x] `pkg/util/file`：读写辅助
- [x] `pkg/notification` + `/notifications/*`；`WebAPI.notification` 开启时 WARN+ 日志转通知并 WS 广播
- [x] `cluster.StartNodeProxyRefresh`（5s，agent `proxy_id` 摘要）

说明：Docker Chrome MITM、Android ADB、Proxy `initVPS` 脚本上传、FileTransporter 后台同步仍为 Java 级能力或运维脚本依赖。

## 风险与决策点（实施前需确认）

| 项 | 说明 |
|----|------|
| Chrome 自动化 | Go 无 Selenium 同等栈：选项 A 保留 Java Chrome 子进程；B chromedp；C 远程 WebDriver 服务 |
| H2 驱动 | Go 使用 `database/sql` + 迁移工具复刻 ORMLite 表结构 |
| JSON 多态 | Template/Builder/Proc 需对齐 Gson `RuntimeTypeAdapterFactory` 行为 |
| 并发模型 | Java 线程 → goroutine + channel，注意 Token 顺序与 `Distributor` 锁语义 |

## 完成定义（Definition of Done）

1. `docs/API_INVENTORY.md` 中每条路由行为与 Java 一致（状态码、body、header）。
2. 前端 `web/` 主要流程无改接口即可运行。
3. Docker Compose 可仅用 Go 二进制替换 Java WebAPI  jar（或并行灰度）。
