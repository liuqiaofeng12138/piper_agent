from __future__ import annotations

import re
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from dataclasses import dataclass
from typing import Callable

ATOM_NS = "http://www.w3.org/2005/Atom"
ARXIV_API = "http://export.arxiv.org/api/query"
DEFAULT_USER_AGENT = "piper-paper-agent/0.1 (academic search; contact: local-dev)"


@dataclass(frozen=True)
class Paper:
    title: str
    summary: str
    authors: list[str]
    published: str
    arxiv_id: str
    url: str


def extract_search_params(user_message: str, max_cap: int = 20) -> tuple[str, int]:
    """从自然语言问题中抽取检索主题与篇数。"""
    msg = (user_message or "").strip()
    if not msg:
        return "", 5

    max_results = 5
    m = re.search(r"(\d+)\s*篇", msg)
    if m:
        max_results = int(m.group(1))

    topic = msg
    patterns = [
        r"关于[「\"']?(.+?)[」\"']?的(?:论文|文献|paper|papers)",
        r"(?:搜索|查询|找|检索)(?:一下)?(?:最新|最近)?(?:的)?(.+?)(?:相关)?(?:论文|文献|paper|papers)",
        r"(?:latest|recent)\s+(\d+)?\s*papers?\s+(?:on|about)\s+(.+)",
    ]
    for pattern in patterns:
        m = re.search(pattern, msg, re.IGNORECASE)
        if not m:
            continue
        groups = [g for g in m.groups() if g]
        if pattern.startswith("(?:latest"):
            if len(groups) == 2 and groups[0].isdigit():
                max_results = int(groups[0])
                topic = groups[1].strip()
            else:
                topic = groups[-1].strip()
        else:
            topic = m.group(1).strip()
        break

    topic = re.sub(
        r"^(请|帮我|帮忙|查询|搜索|找|检索|汇总|展示|列出|给我|最新的|最近的)\s*",
        "",
        topic,
        flags=re.IGNORECASE,
    )
    topic = re.sub(r"\s+", " ", topic).strip(" ，。,.")
    if not topic:
        topic = msg

    cap = max(1, max_cap)
    max_results = max(1, min(max_results, cap))
    return topic, max_results


def _arxiv_search_query(topic: str) -> str:
    """将用户主题转为 arXiv search_query（all: 字段，空格用 AND 连接）。"""
    tokens = re.findall(r"[\w.+#/-]+", topic, flags=re.UNICODE)
    if not tokens:
        return f"all:{urllib.parse.quote(topic)}"
    parts = [f"all:{urllib.parse.quote(t)}" for t in tokens[:12]]
    return " AND ".join(parts)


def search_arxiv(
    topic: str,
    max_results: int,
    *,
    timeout_seconds: float = 30.0,
    fetch: Callable[[str, float], bytes] | None = None,
) -> list[Paper]:
    """调用 arXiv Atom API 按提交时间倒序检索论文。"""
    topic = (topic or "").strip()
    if not topic:
        return []

    query = _arxiv_search_query(topic)
    params = urllib.parse.urlencode(
        {
            "search_query": query,
            "start": 0,
            "max_results": max(1, max_results),
            "sortBy": "submittedDate",
            "sortOrder": "descending",
        }
    )
    url = f"{ARXIV_API}?{params}"

    def _default_fetch(u: str, timeout: float) -> bytes:
        req = urllib.request.Request(u, headers={"User-Agent": DEFAULT_USER_AGENT})
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.read()

    raw = (fetch or _default_fetch)(url, timeout_seconds)
    return _parse_arxiv_atom(raw)


def _parse_arxiv_atom(data: bytes) -> list[Paper]:
    root = ET.fromstring(data)
    papers: list[Paper] = []
    for entry in root.findall(f"{{{ATOM_NS}}}entry"):
        title = _text(entry, "title")
        summary = _text(entry, "summary")
        published = _text(entry, "published")
        arxiv_id = _text(entry, "id")
        link = ""
        for link_el in entry.findall(f"{{{ATOM_NS}}}link"):
            if link_el.get("rel") == "alternate" and link_el.get("href"):
                link = link_el.get("href") or ""
                break
        if not link and arxiv_id:
            link = arxiv_id

        authors: list[str] = []
        for author in entry.findall(f"{{{ATOM_NS}}}author"):
            name = _text(author, "name")
            if name:
                authors.append(name)

        papers.append(
            Paper(
                title=title,
                summary=summary,
                authors=authors,
                published=published[:10] if published else "",
                arxiv_id=_short_arxiv_id(arxiv_id),
                url=link,
            )
        )
    return papers


def _text(parent: ET.Element, local: str) -> str:
    el = parent.find(f"{{{ATOM_NS}}}{local}")
    if el is None or el.text is None:
        return ""
    return re.sub(r"\s+", " ", el.text).strip()


def _short_arxiv_id(arxiv_url: str) -> str:
    if not arxiv_url:
        return ""
    m = re.search(r"arxiv\.org/abs/(.+)$", arxiv_url)
    if m:
        return m.group(1)
    return arxiv_url.rsplit("/", 1)[-1]


def format_papers_context(papers: list[Paper]) -> str:
    if not papers:
        return "（未检索到相关论文。）"
    blocks: list[str] = []
    for i, p in enumerate(papers, start=1):
        authors = ", ".join(p.authors[:8])
        if len(p.authors) > 8:
            authors += " 等"
        abstract = p.summary
        if len(abstract) > 1200:
            abstract = abstract[:1200] + "…"
        blocks.append(
            f"### [{i}] {p.title}\n"
            f"- arXiv: {p.arxiv_id}\n"
            f"- 发布日期: {p.published}\n"
            f"- 作者: {authors or '未知'}\n"
            f"- 链接: {p.url}\n"
            f"- 摘要: {abstract}"
        )
    return "\n\n".join(blocks)
