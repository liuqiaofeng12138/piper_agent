# internal/svc

`ServiceContext`：依赖注入容器，供所有 `logic` 使用。

## 应持有的依赖（随阶段增加）

| 字段类型 | 用途 | Java 参考 |
|----------|------|-----------|
| `config.Config` | 配置 | — |
| ES / S3 客户端 | 存储 | `ESClient`, `S3Adapter` |
| H2 DAO | 元数据 | `Daos`, `ModelD` |
| `*distributor.Distributor` | 调度 | `ChromeDistributor`, `HttpDistributor` |
| UserCache | 鉴权 | `Caches.userCache` |
| Node 本地信息 | `/misc/info` | `WebAPI.node` |
| Scheduler | 定时任务 | `Scheduler` |
| FileTransporter | 迁移 | `FileTransporter.getInstance()` |
| WebSocket Hub | 广播 | `MsgPublisher` |

## 约定

- 仅 **构造与 Getter**，不写业务方法。
- 在 `NewServiceContext(c config.Config)` 中完成客户端初始化（或 lazy init 由 bootstrap 触发）。

## 现有文件

- `servicecontext.go` — goctl 生成，需扩展字段。
