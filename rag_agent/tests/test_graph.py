from rag_agent.rag.graph import build_rag_graph
from rag_agent.rag.store import DocumentStore


def test_langgraph_ingest_and_retrieve():
    store = DocumentStore()
    graph = build_rag_graph(store, top_k=3)
    out = graph.invoke(
        {
            "conversation_id": "conv-1",
            "question": "合同金额",
            "new_documents": [("deal.docx", "本合同金额为人民币一百万元整。")],
        }
    )
    assert out.get("ingested_count", 0) >= 1
    assert "一百万元" in (out.get("retrieved_context") or "")
