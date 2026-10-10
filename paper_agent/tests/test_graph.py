from paper_agent.papers.graph import build_paper_search_graph

SAMPLE_ATOM = b"""<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00002</id>
    <title>Graph Test Paper</title>
    <summary>Abstract text.</summary>
    <published>2024-02-01T00:00:00Z</published>
    <author><name>Carol</name></author>
    <link rel="alternate" href="http://arxiv.org/abs/2401.00002"/>
  </entry>
</feed>
"""


def test_graph_invoke_with_mocked_search(monkeypatch):
    from paper_agent.papers import graph as graph_mod
    from paper_agent.papers import search as search_mod

    def fake_search(topic, max_results, timeout_seconds=30.0, fetch=None):
        return search_mod._parse_arxiv_atom(SAMPLE_ATOM)

    monkeypatch.setattr(graph_mod, "search_arxiv", fake_search)
    graph = build_paper_search_graph(max_results_cap=10, arxiv_timeout_seconds=5.0)
    out = graph.invoke({"user_message": "找3篇关于 graph 的论文"})
    assert out["max_results"] == 3
    assert "graph" in out["search_topic"].lower()
    assert len(out["papers"]) == 1
    assert "Graph Test Paper" in out["papers_context"]
