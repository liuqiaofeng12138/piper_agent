# internal/bootstrap

服务 **启动编排**，对齐 Java `WebAPI.main` 与 `buildHttpApiServer`。

## 应实现的流程（顺序参考 Java）

1. 加载配置（`internal/config`）。
2. 等待 ES/S3 就绪（`pkg/bootstrap` / `InitUtil.waitDatabaseReady` 等价）。
3. 设置 `ready=true`（与 middleware Ready 检查联动）。
4. 首次运行：CA/SSH 密钥、`initTables`、`initDefaultIndices`、S3 bucket（Vars）。
5. `Cache.init`、恢复 Agents、`syncTasks`。
6. 注册 Distributor：Chrome + Http，`Persister` / `TokenPersister` / 回调。
7. 启动定时任务：Prometheus 节点同步、本机 proxy 信息、agents 写盘。
8. `FileTransporter`、Notification 可选初始化。
9. 注册 HTTP routes + WebSocket（或由 `pipergo.go` 调用本包 `MustSetup()`）。

## 代码类型

- `Setup(ctx context.Context, svc *svc.ServiceContext) error`
- `Shutdown()` 优雅关闭 scheduler、agent、distributor

## Java 对照

- [WebAPI.java](../../src/main/java/one/rewind/nio/web/WebAPI.java)
- [InitUtil.java](../../src/main/java/one/rewind/nio/util/InitUtil.java)
