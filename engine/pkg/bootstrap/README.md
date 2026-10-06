# pkg/bootstrap

启动期 **非 HTTP** 初始化逻辑，对齐 Java `InitUtil` 中与 Web 无关的部分。

## 已实现

- `EnsureDependencies` — `requireDeps=true` 时检查 Docker 容器（`requiredContainers`）及 ES/S3/Prometheus 端点，超时后启动失败
- `EnsureGrafanaDashboard` — Grafana 可达但缺少 `MM3UsuH7z` 仪表盘时，从 `docker/grafana/.../piper-dashboard.json` 导入（需 `grafanaAdminUser` / `grafanaAdminPassword`）
- `WaitStorageReady(es, s3)` — `requireDeps=false` 时沿用（可由 `skipStorageWait` 跳过）

## 应实现

- `WaitDatabaseReady(es, s3)`
- `InitTables()` / `H2TablesReady()`
- `InitDefaultIndices()`
- `ConfigOverlay(cfg)` — 覆盖 ES/S3/Requester 等运行时配置
- `RestoreAgents()`, `SyncTasks()`
- `RequireGenerateKeys(path)` — certs、pk 目录

## Java 对照

`src/main/java/one/rewind/nio/util/InitUtil.java`

