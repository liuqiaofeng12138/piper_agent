from paper_agent.papers.search import (
    _parse_arxiv_atom,
    extract_search_params,
    format_papers_context,
    search_arxiv,
)

SAMPLE_ATOM = b"""<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2401.00001</id>
    <title>AI Infra at Scale</title>
    <summary>We study AI infrastructure.</summary>
    <published>2024-01-02T00:00:00Z</published>
    <author><name>Alice</name></author>
    <author><name>Bob</name></author>
    <link rel="alternate" href="http://arxiv.org/abs/2401.00001"/>
  </entry>
</feed>
"""


def test_extract_search_params_chinese():
    topic, n = extract_search_params("查询最新的5篇关于ai infra的论文，汇总展示给我")
    assert n == 5
    assert "ai infra" in topic.lower()


def test_parse_arxiv_atom():
    papers = _parse_arxiv_atom(SAMPLE_ATOM)
    assert len(papers) == 1
    assert papers[0].title == "AI Infra at Scale"
    assert papers[0].arxiv_id == "2401.00001"
    assert papers[0].authors == ["Alice", "Bob"]


def test_search_arxiv_with_mock_fetch():
    def fetch(_url: str, _timeout: float) -> bytes:
        return SAMPLE_ATOM

    papers = search_arxiv("ai infra", 3, fetch=fetch)
    assert len(papers) == 1
    ctx = format_papers_context(papers)
    assert "AI Infra at Scale" in ctx
