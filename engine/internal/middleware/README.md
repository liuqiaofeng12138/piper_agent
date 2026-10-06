# internal/middleware

HTTP 中间件，对齐 Spark `Routes.init` 中 `before` / `after` / `options`。

## 应实现

| 中间件 | Java 行为 |
|--------|-----------|
| CORS | `OPTIONS /*`；允许 Origin/Methods/Headers；暴露 `Authorization` |
| ReadyGuard | 非 InitRoutes 且未 ready → 500 `Server not ready` |
| Auth | `noAuth=false` 时校验 token（除 NoAuthRoutes、OPTIONS）→ 401 `Token error` |
| ResponseHeaders | JSON 默认头、gzip、`Cache-Control: no-store`；`/metrics` 例外 |
| AdminRole | 路径匹配 AdminRoutes 时校验 Admin 角色 |
| RequestTiming | 可选：记录 begin_ts（Java session attribute） |

## Java 对照

- [Routes.java](../../src/main/java/one/rewind/nio/web/route/Routes.java) — `NoAuthRoutes`, `InitRoutes`, `WsRoutes`
- [Authenticator.java](../../src/main/java/one/rewind/nio/web/filter/Authenticator.java)

## 依赖

- `pkg/auth` — token 解析
- `internal/svc` — UserCache、配置 `NoAuth`
