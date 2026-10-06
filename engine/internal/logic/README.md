# internal/logic

**用例层**：编排 `pkg/*` 完成单个 API 或组合操作，对齐 Java `*Route` 中的静态方法逻辑。

## 组织方式

子目录与 `internal/handler` **同名一一对应**（见 handler README 表）。

每个 logic 文件对应 goctl 生成的 `*Logic` 结构体，持有 `context.Context` 与 `*svc.ServiceContext`。

## 约定

- 事务边界、ES/S3/H2 调用、Distributor 提交在此层完成。
- 返回业务 error 时映射为与 Java 一致的 HTTP 状态与 message（参考 `JsonTransformer` / halt 行为）。
- 复杂可复用逻辑下沉到 `pkg/`，logic 只做编排。

## 测试

优先为 logic 编写表驱动测试，对照 Java `src/test/java/one/rewind/nio/web/test/*RouteTest.java`。
