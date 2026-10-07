# tools

每个 Tool 对应一组 gRPC 调用或本地逻辑（Prompt 片段加载）。

## 首批工具

| Tool | Runtime RPC | 说明 |
|------|-------------|------|
| `list_templates` | `RuntimeMeta.ListTemplates` | RAG 前先列库 |
| `get_template` | `RuntimeMeta.GetTemplate` | 克隆现有模版 |
| `validate_template` | `RuntimeTemplate.ValidateTemplate` | 强制门禁 |
| `upsert_template` | `RuntimeTemplate.UpsertTemplate` | 写入 meta |
| `run_template` | `RuntimeExecute.RunTemplate` | 执行 |
| `get_run_status` | `RuntimeExecute.SubscribeRun` / Get | 轮询或 stream |
| `get_token_data` | `RuntimeData.GetTokenData` | 取 Mapper 结果 |
| `list_proxies` | `RuntimeMeta.ListProxies` | 选代理 |

## 实现约定

- 使用 `clients/runtime_client.py` 单例 channel
- Tool 返回 JSON 字符串供 LLM 消费，错误带 `diagnostics[]`
