# Java → Go 包与目录映射

Java 根路径：`src/main/java/one/rewind/`  
Go 根路径：`engine/`（模块名 `piper_go`）

## 顶层映射

| Java 包 | Go 目录 | 说明 |
|---------|---------|------|
| `one.rewind.nio.web` | `internal/bootstrap`, `internal/middleware`, `pipergo.go` | HTTP 服务生命周期 |
| `one.rewind.nio.web.route.*` | `api/*`, `internal/handler/*`, `internal/logic/*` | REST 路由 |
| `one.rewind.nio.web.filter` | `internal/middleware` | 鉴权 |
| `one.rewind.nio.web.auth` | `pkg/auth` | 登录/Token |
| `one.rewind.nio.web.model` | `pkg/db/model` + types | User 等 |
| `one.rewind.nio.web.cache` | `internal/svc` 或 `pkg/distributor` 缓存 | UserCache |
| `one.rewind.nio.web.serialization` | `pkg/json` + handler 响应封装 | JsonTransformer, Msg |
| `one.rewind.nio.web.websocket` | `pkg/websocket` | WS 发布 |
| `one.rewind.nio.web.notification` | `pkg/notification` | 通知模型与推送 |
| `one.rewind.db.*` | `pkg/db/*` | ES、S3、H2 Model、Kafka |
| `one.rewind.nio.util.InitUtil` | `pkg/bootstrap`, `internal/bootstrap` | 初始化表、默认 Index、Agent 恢复 |
| `one.rewind.nio.distributor.*` | `pkg/distributor/*` | Token、Agent 调度、Stats |
| `one.rewind.nio.chrome.*` | `pkg/agent/chrome`, `pkg/chrome/*` | Chrome Agent、动作、阶段 |
| `one.rewind.nio.http.*` | `pkg/agent/http`, `pkg/http` | HttpAgent、Requester |
| `one.rewind.nio.android.*` | `pkg/agent/android` | Android Agent |
| `one.rewind.nio.proxy.*` | `pkg/proxy/*` | 代理模型与 PPPD |
| `one.rewind.nio.task.*` | `pkg/task` | Task、Scheduler、VarsList、Pack |
| `one.rewind.nio.tpl.*` | `pkg/tpl/*` | 模板引擎、Builder、Mapper、Proc |
| `one.rewind.nio.func` | `pkg/func` | 可复用函数实体 |
| `one.rewind.nio.index` | `pkg/index` | Index 元数据与 Doc |
| `one.rewind.nio.account.*` | `pkg/account` | 账号 CRUD |
| `one.rewind.nio.persistence.*` | `pkg/persistence/*` | 采集结果落库、Source、迁移 |
| `one.rewind.nio.cluster.*` | `pkg/cluster` | 节点、RemoteRequester、FileTransporter |
| `one.rewind.nio.docker.*` | `pkg/docker` | Chrome 容器、DockerHost |
| `one.rewind.nio.json.*` | `pkg/json/*` | 与 Java Gson 行为对齐的序列化 |
| `one.rewind.nio.cert` | `pkg/cert` | MITM/CA 证书 |
| `one.rewind.monitor.*` | `pkg/monitor/*` | 主机指标（info、metrics 辅助） |
| `one.rewind.network.*` | `pkg/network/*` | SSH、端口探测 |
| `one.rewind.util.*` | `pkg/util` | 配置、文件、密钥、网络工具 |
| `one.rewind.txt.*` | `pkg/txt` | 文本清洗、日期、相似度 |
| `one.rewind.simulator.*` | `pkg/simulator/mouse` | 鼠标轨迹模拟 |
| `one.rewind.skeleton.*` | `pkg/skeleton` | 并发/分区工具 |
| `one.rewind.log` | `pkg/log` | ES Log Appender |

## H2 / ModelD 表（InitUtil.initTables）

需在 `pkg/db/h2` 实现与 Java ORMLite 等价的实体与迁移：

| Java 类 | 用途 |
|---------|------|
| `Index` | 索引定义 |
| `Template` | 采集模板 |
| `ProxyPPPD` | WireGuard/PPP 代理 |
| `AccountImpl` | 账号 |
| `Task` | 任务 |
| `VarsList` | 变量列表 |
| `Func` | 函数库 |
| `NodeInfo` | 集群节点 |
| `User` | 用户 |

ES 动态 Index（默认）：`Author`, `Comment`, `Essay`, `Media`, `Post`, `Movie`（见 `Defaults` / `initDefaultIndices`）。

## Distributor 核心类

| Java | Go 目标 |
|------|---------|
| `Distributor` | `pkg/distributor/distributor.go` |
| `Agent` / `ChromeAgent` / `HttpAgent` | `pkg/agent/*` |
| `Token` | `pkg/distributor/token.go` |
| `Cache` | `pkg/distributor/cache.go` |
| `Stats` | `pkg/distributor/stats.go`（对接 `/metrics`） |
| `Proc` | `pkg/tpl/proc.go` |
| `ChromeDistributor` / `HttpDistributor` | `pkg/distributor/chrome.go`, `http.go` |
| `Persister`, `TokenPersister`, `TokenDependencyResolver` | `pkg/distributor/processing/*` |
| `*Callback` | `pkg/distributor/callback/*` |
| `*Exception` | `pkg/distributor/exception/*` |

## Template / Proc 类型

| Java | 说明 |
|------|------|
| `Builder`（Http/Chrome/…） | 模板构建器 |
| `Mapper`, `If`, `For`, `Interceptor` | 流程节点 |
| `Action` 及 chrome/tpl action | 执行步骤 |
| `Parser`, `ValidationUtil`, `Vars` | 解析与变量 |

## 配置文件对照

| Java | Go |
|------|-----|
| `conf.sample/WebAPI.conf` | `etc/pipergo-api.yaml` 中 WebAPI 段 |
| `conf.sample/ESClient.conf` | `ES` 段 |
| `conf.sample/S3Adapter.conf` | `S3` 段 |
| `conf.sample/Auth.conf` | `Auth` 段 |
| `conf.sample/Requester.conf` | `Requester` 段 |
| `conf.sample/KafkaClient.conf` | `pkg/db/kafka` |
| 其他 Adapter conf | 按需在 config 扩展 |

## 测试对照

Java 集成测试目录 → Go 建议 `tests/integration/` 或 `internal/logic/*/*_test.go`：

- `src/test/java/one/rewind/nio/web/test/*RouteTest.java`
- `src/test/java/one/rewind/nio/task/test/*`
- `src/test/java/one/rewind/nio/tpl/test/*`

## 不在本次 HTTP 服务内但需规划的包

| 组件 | 说明 |
|------|------|
| `__mitm-proxy/` | 独立 MITM 代理工程，Go 版 Piper 通过 HTTP/配置与之协作 |
| Chrome Docker 镜像 | 仍由 `docker/chrome` 构建，Go 通过 Docker API 管理 |
| 前端 `web/` | 仅消费 API，不迁入 `piper_go` |
