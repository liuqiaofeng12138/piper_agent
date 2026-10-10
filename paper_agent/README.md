# paper_agent

学术论文搜索子 Agent：网关路由到 `paper_search` Worker，经 **LangGraph** 解析意图并 **联网检索 arXiv**，再由大模型流式汇总展示。

## 能力

- 自然语言指定主题与篇数（如「查询最新的 5 篇关于 ai infra 的论文」）
- arXiv API 按提交时间倒序检索
- 与 `doc_rag` 相同 gRPC 契约：`agent.v1.AgentWorkerService`

## 本地运行 Worker

```powershell
cd paper_agent
..\.venv\Scripts\python.exe -m paper_agent.workers.paper_search --config ..\deploy\config\local.yaml
```

推荐通过 `piper-serve` 一键拉起（需在 `deploy/config/local.yaml` 的 `agents` 中启用 `paper_search`）。

## 配置

`deploy/config/local.yaml`：

```yaml
agents:
  - id: paper_search
    listen: "127.0.0.1:15064"
    enabled: true
    module: paper_agent.workers.paper_search

paper_search:
  max_results_cap: 20
  arxiv_timeout_seconds: 30
```

复用根配置中的 `llm` 段做汇总生成。

## 测试

```powershell
cd paper_agent
.\.venv\Scripts\python.exe -m pytest tests/
```
