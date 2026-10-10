from __future__ import annotations

from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from rag_agent.rag.store import DocumentStore


class RAGGraphState(TypedDict, total=False):
    conversation_id: str
    question: str
    new_documents: list[tuple[str, str]]
    ingested_count: int
    retrieved_context: str
    filenames: list[str]


def build_rag_graph(store: DocumentStore, top_k: int):
    """LangGraph：入库 → 检索，生成由 Worker 流式调用 LLM。"""

    def ingest_node(state: RAGGraphState) -> dict:
        conv = state["conversation_id"]
        added = 0
        for filename, text in state.get("new_documents") or []:
            added += store.add_document(conv, filename, text)
        return {"ingested_count": added, "filenames": store.list_filenames(conv)}

    def retrieve_node(state: RAGGraphState) -> dict:
        ctx_chunks = store.retrieve(state["conversation_id"], state.get("question") or "", top_k=top_k)
        if ctx_chunks:
            context = "\n\n---\n\n".join(ctx_chunks)
        else:
            context = "（当前会话尚无文档内容，请先上传 PDF/Word 文件。）"
        return {"retrieved_context": context}

    graph = StateGraph(RAGGraphState)
    graph.add_node("ingest", ingest_node)
    graph.add_node("retrieve", retrieve_node)
    graph.add_edge(START, "ingest")
    graph.add_edge("ingest", "retrieve")
    graph.add_edge("retrieve", END)
    return graph.compile()
