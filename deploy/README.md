# deploy

本地与容器部署。

## 配置

| 文件 | 说明 |
|------|------|
| `config/local.yaml` | **统一配置**：gRPC Runtime、`engine` 段（ES/S3/meta/Chrome）、Python Agent（`runtime`/`llm`/`harness`） |
| `config/local.yaml.example` | 模板（复制为 `local.yaml` 后填写 `llm.api_key`） |

`local.yaml` 已加入 `.gitignore`，避免误提交密钥。

## 拓扑

```
[User] → piper-agent (Python) --gRPC--> piper-runtime (Go) --> engine (piper_go) --> ES/S3
```
