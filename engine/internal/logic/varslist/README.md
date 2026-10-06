# internal/logic/varslist

## 职责

用例编排：实现本域 API 业务，调用 pkg 层。

## 覆盖接口

CRUD /vars_lists, check, first

## Java 对照

- `src/main/java/one/rewind/nio/web/route/VarsListRoute.java`
- 路由注册：`Routes.java`（`initNodeRoutes` / `initClientRoutes`）

## 依赖 pkg（典型）

- pkg/task (VarsList), pkg/db/h2, pkg/db/s3

## API 文档

见 [docs/API_INVENTORY.md](../../../docs/API_INVENTORY.md) 对应章节。

