# internal/handler/account

## 职责

HTTP 接入：绑定路由、调用 logic、设置 status/header。

## 覆盖接口

/accounts*, filter, domains, usernames

## Java 对照

- `src/main/java/one/rewind/nio/web/route/AccountRoute.java`
- 路由注册：`Routes.java`（`initNodeRoutes` / `initClientRoutes`）

## 依赖 pkg（典型）

- pkg/account, pkg/db/h2

## API 文档

见 [docs/API_INVENTORY.md](../../../docs/API_INVENTORY.md) 对应章节。

