# internal/handler/template

## 职责

HTTP 接入：绑定路由、调用 logic、设置 status/header。

## 覆盖接口

CRUD /templates, mapper_types, run, ref, var_names

## Java 对照

- `src/main/java/one/rewind/nio/web/route/TemplateRoute.java`
- 路由注册：`Routes.java`（`initNodeRoutes` / `initClientRoutes`）

## 依赖 pkg（典型）

- pkg/tpl, pkg/distributor

## API 文档

见 [docs/API_INVENTORY.md](../../../docs/API_INVENTORY.md) 对应章节。

