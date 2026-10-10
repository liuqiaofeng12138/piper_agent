from __future__ import annotations

from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from paper_agent.papers.search import (
    Paper,
    extract_search_params,
    format_papers_context,
    search_arxiv,
)


class PaperGraphState(TypedDict, total=False):
    user_message: str
    search_topic: str
    max_results: int
    papers: list[Paper]
    papers_context: str
    search_error: str


def build_paper_search_graph(max_results_cap: int, arxiv_timeout_seconds: float):
    """LangGraph：解析意图 → arXiv 检索 → 组装供 LLM 汇总的上下文。"""

    def parse_node(state: PaperGraphState) -> dict:
        topic, count = extract_search_params(state.get("user_message") or "", max_results_cap)
        return {"search_topic": topic, "max_results": count}

    def search_node(state: PaperGraphState) -> dict:
        topic = state.get("search_topic") or ""
        count = int(state.get("max_results") or 5)
        try:
            papers = search_arxiv(topic, count, timeout_seconds=arxiv_timeout_seconds)
            return {"papers": papers, "search_error": ""}
        except Exception as e:  # noqa: BLE001
            return {"papers": [], "search_error": str(e)}

    def prepare_node(state: PaperGraphState) -> dict:
        err = (state.get("search_error") or "").strip()
        papers = state.get("papers") or []
        if err:
            ctx = f"（联网检索失败: {err}）"
        else:
            ctx = format_papers_context(papers)
        return {"papers_context": ctx}

    graph = StateGraph(PaperGraphState)
    graph.add_node("parse", parse_node)
    graph.add_node("search", search_node)
    graph.add_node("prepare", prepare_node)
    graph.add_edge(START, "parse")
    graph.add_edge("parse", "search")
    graph.add_edge("search", "prepare")
    graph.add_edge("prepare", END)
    return graph.compile()
