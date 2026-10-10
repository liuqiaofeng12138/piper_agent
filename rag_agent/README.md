# rag_agent（Python）

**文档 RAG 子 Agent**：Web 上传 PDF/Word → 文本抽取 → LangGraph（入库 + 检索）→ LLM 流式问答。  
由网关 `gateway/` 经 gRPC `agent.v1.AgentWorkerService` 调用；**附带文件时网关跳过意图识别，固定路由到 `doc_rag`**。

## 安装

```bash
cd rag_agent
python -m venv .venv
.venv\Scripts\pip install -e ".[dev]"   # Windows（含 markitdown 的 PDF/Word 解析依赖）
```

## 单独调试 Worker

```powershell
python -m rag_agent.workers.doc_rag --config ..\deploy\config\local.yaml --listen 127.0.0.1:15063
```

通常由 `piper-serve` 按 `deploy/config/local.yaml` 的 `agents` 注册表自动拉起。

## LangGraph 流程

```
ingest（新文档分块入库） → retrieve（关键词检索 Top-K） → Worker 流式 LLM 生成
```

会话级知识库保存在 Worker 进程内存中，重启后需重新上传文档。
