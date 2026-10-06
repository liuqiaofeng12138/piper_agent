# etc

运行时数据目录（非配置文件）。

| 路径 | 说明 |
|------|------|
| `db/raw_meta.db` | meta SQLite（模版、代理等） |
| `db/agents_info.json` | Agent 注册快照 |

采集内核的连接与 Chrome 等参数在仓库根 **`deploy/config/local.yaml`** 的 `engine:` 段配置；`engine.H2.path` 通常指向本目录下的 `db/raw_meta.db`（相对 `deploy/config/` 解析）。
