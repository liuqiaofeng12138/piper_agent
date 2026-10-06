# 运行时配置

存放 go-zero 与服务专用 YAML/JSON。

## 文件

| 文件 | 说明 |
|------|------|
| `pipergo-api.yaml` | REST 服务主配置（Host、Port、日志、超时） |

## 需扩展的配置段（对齐 Java `WebAPI.conf` 与 `conf.sample/*`）

| 配置段 | Java 参考 | 用途 |
|--------|-----------|------|
| `WebAPI` | `WebAPI.conf` | noAuth、notification、onlyVideo、prometheusHost、超时 |
| `WebAPI.requireDeps` | — | 为 `true` 时启动前检查 `requiredContainers`（Docker 运行/健康）及 ES/S3/Prometheus 可达，失败则退出 |
| `ES` | `ESClient.conf` | Elasticsearch 地址 |
| `S3` | `S3Adapter.conf` | MinIO/S3 端点与密钥 |
| `Auth` | `Auth.conf` | JWT/Session 密钥、过期时间 |
| `Requester` | `Requester.conf` | HTTP 客户端限速与超时 |
| `H2` | `PooledDataSource.conf` | 嵌入式元数据库路径 |
| `Kafka` | `KafkaClient.conf` | 可选消息队列 |
| `FileTransporter` | `FileTransferAutomator.conf` | 数据迁移 MySQL/FTP |
| `Paths` | 代码常量 | `agents`、`cache/fingerprints.ser`、`pk/`、`certs/` |

## 约定

- 敏感项走环境变量覆盖，不提交密钥。
- 生产默认 **Port: 81** 与 Java Spark 一致（开发可用 8888）。
