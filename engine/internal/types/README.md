# internal/types

goctl 根据 `api/*.api` **自动生成** 的请求/响应结构体。

## 约定

- 手改需谨慎：重新 goctl 可能覆盖；自定义类型可放 `pkg/` 或在 `.api` 中使用 `type` 别名。
- 与 Java REST body/query 字段名保持一致（含 json tag），便于前端零改动。
- 分页、统一错误体可在 `api/base.api` 定义后生成到此目录。

## 禁止

- 不在 types 中写业务方法或调用 ES/S3。

## 参考

Java 各 `*Route.java` 中解析的 Gson 类型、`Routes.QueryInfo` 查询参数。
