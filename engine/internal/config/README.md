# internal/config

go-zero 配置结构体定义，由 `etc/*.yaml` 加载。

## 应实现的类型

- 嵌入 `rest.RestConf`（已有）。
- 扩展字段映射 Java `Configs.getConfig(WebAPI.class)` 及 ES/S3/Auth/Requester 等（见 [etc/README.md](../../etc/README.md)）。
- 可选：`Validate()` 在启动时检查必填项。

## Java 对照

- `one.rewind.util.Configs`
- `com.typesafe.config.Config` 在 `WebAPI.main` / `InitUtil.configOverlay` 中的用法

## 禁止

- 不在 config 包写业务逻辑或访问 ES/S3。
