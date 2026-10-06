# 规划文档

本目录存放 **Go 重写 Piper** 的对照与计划文档，不写业务代码。

| 文件 | 用途 |
|------|------|
| [API_INVENTORY.md](API_INVENTORY.md) | Java `Routes.java` 导出的完整 REST/WebSocket 清单 |
| [JAVA_MAPPING.md](JAVA_MAPPING.md) | Java 包/类 → `piper_go` 目录映射 |
| [IMPLEMENTATION_PHASES.md](IMPLEMENTATION_PHASES.md) | 分阶段实施顺序、依赖关系、风险点 |

更新约定：接口或 Java 行为变更时，先更新此处再改 `api/` 与实现。
