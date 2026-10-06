# deploy

本地与容器部署。

## 文件（规划）

| 文件 | 说明 |
|------|------|
| `docker-compose.agent.yaml` | runtime + agents CLI 容器；依赖 ES/S3 可指向现有 `docker/piper_dev.yaml` |
| `config/runtime.local.yaml` | gRPC 监听、`engine/etc/pipergo-api.yaml` 路径 |
| `config/agent.local.yaml` | Runtime 地址、`llm.api_key`、模型名（无需环境变量） |
| `config/agent.local.yaml.example` | 配置模板与说明 |
| `config/agent.secrets.yaml` | 可选，仅覆盖 `llm.api_key`（gitignore） |

## 拓扑

```
[User] → piper-agent (Python) --gRPC--> piper-runtime (Go) --> engine (piper_go) --> ES/S3
```
