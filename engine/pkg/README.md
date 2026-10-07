# pkg — 领域与基础设施

可被 `internal/logic`、`internal/bootstrap` 引用的 **核心业务库**（Go 惯例：对外可复用、不依赖 handler）。

## 模块索引

| 目录 | Java 包 | 职责摘要 |
|------|---------|----------|
| [bootstrap](bootstrap/README.md) | `InitUtil` 部分 | 等存储就绪、建表、默认数据 |
| [db](db/README.md) | `one.rewind.db` | ES、S3、H2、Kafka、Model 基类 |
| [distributor](distributor/README.md) | `nio.distributor` | Token 调度、Stats、Cache |
| [agent](agent/README.md) | chrome/http/android Agent | 执行体 |
| [chrome](chrome/README.md) | `nio.chrome` | 浏览器动作、阶段、过滤 |
| [http](http/README.md) | `nio.http` | HTTP 采集、Requester |
| [proxy](proxy/README.md) | `nio.proxy` | 代理实体与 PPPD |
| [task](task/README.md) | `nio.task` | Task、Scheduler、VarsList、Pack |
| [tpl](tpl/README.md) | `nio.tpl` | 模板 DSL 与执行 |
| [index](index/README.md) | `nio.index` | Index 元数据、Doc、Field |
| [func](func/README.md) | `nio.func` | Func 实体与运行 |
| [account](account/README.md) | `nio.account` | 账号 |
| [persistence](persistence/README.md) | `nio.persistence` | 采集结果、迁移、Source |
| [cluster](cluster/README.md) | `nio.cluster` | 节点、远程调用、FileTransporter |
| [docker](docker/README.md) | `nio.docker` | 容器生命周期 |
| [json](json/README.md) | `nio.json` | 序列化与 Java Gson 对齐 |
| [auth](auth/README.md) | `nio.web.auth` | 登录、JWT |
| [notification](notification/README.md) | `nio.web.notification` | 通知持久化 |
| [websocket](websocket/README.md) | `nio.web.websocket` | WS Hub |
| [monitor](monitor/README.md) | `monitor` | 系统信息采集 |
| [network](network/README.md) | `network` | SSH、探测 |
| [util](util/README.md) | `util` | 文件、配置、密钥 |
| [txt](txt/README.md) | `txt` | 文本工具 |
| [simulator](simulator/README.md) | `simulator` | 鼠标模拟 |
| [skeleton](skeleton/README.md) | `skeleton` | 并发辅助 |
| [cert](cert/README.md) | `nio.cert` | 证书 |
| [log](log/README.md) | `log` | ES 日志 |

## 依赖方向

```
tpl, agent → distributor → persistence → db
task → distributor, tpl
cluster → db, persistence
```

禁止 `pkg/*` import `internal/*`。
