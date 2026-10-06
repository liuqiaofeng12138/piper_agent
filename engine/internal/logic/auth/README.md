# internal/logic/auth

## 职责

用例编排：实现本域 API 业务，调用 pkg 层。

## 覆盖接口

POST /auth/login, POST /auth/logout, GET /auth/me

## Java 对照

- `src/main/java/one/rewind/nio/web/route/AuthRoute.java`
- 路由注册：`Routes.java`（`initNodeRoutes` / `initClientRoutes`）

## 依赖 pkg（典型）

- pkg/auth, pkg/db/model (User)

## API 文档

见 [docs/API_INVENTORY.md](../../../docs/API_INVENTORY.md) 对应章节。

