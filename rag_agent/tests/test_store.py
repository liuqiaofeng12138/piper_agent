from rag_agent.rag.store import DocumentStore


def test_retrieve_prefers_matching_terms():
    store = DocumentStore(chunk_size=200, chunk_overlap=0)
    store.add_document("c1", "a.txt", "Elasticsearch 用于全文检索与日志分析。")
    store.add_document("c1", "b.txt", "今天天气晴朗，适合出门散步。")
    hits = store.retrieve("c1", "Elasticsearch 检索", top_k=2)
    assert hits
    assert "Elasticsearch" in hits[0]
