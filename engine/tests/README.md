# tests

集成与端到端测试（规划中，本阶段仅目录约定）。

## 建议结构

| 路径 | 说明 |
|------|------|
| `tests/integration/route/` | 按域对照 Java `*RouteTest.java` |
| `tests/integration/tpl/` | 模板执行抽样（Java `nio/tpl/test`） |
| `tests/fixtures/` | JSON 样例、最小 H2/ES 数据 |

## 运行约定

- 依赖 Docker Compose 拉起 ES、MinIO、H2（与 Java 开发环境一致）。
- CI 中可先跑不依赖 Chrome 的 CRUD + HTTP Agent 用例。

## 验收基准

[docs/API_INVENTORY.md](../docs/API_INVENTORY.md) + 现有 Java 测试类。
