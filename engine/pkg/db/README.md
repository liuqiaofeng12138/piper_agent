# pkg/db

Java `one.rewind.db` 的 Go 实现入口。

## 子目录

| 目录 | Java 类 |
|------|---------|
| [es](es/README.md) | `ESClient` |
| [s3](s3/README.md) | `S3Adapter` |
| [h2](h2/README.md) | `PooledDataSource`, `Daos`, `Refactor` |
| [kafka](kafka/README.md) | `KafkaClient` |
| [model](model/README.md) | `Model`, `ModelD`, `ModelL` |

## 职责

- 连接池与实例名（`es` / `s3` / `ts` 与 Java 一致）
- Index 命名、Doc 序列化（`DocSerializer`）
- 异常：`DBInitException`, `ModelException`

