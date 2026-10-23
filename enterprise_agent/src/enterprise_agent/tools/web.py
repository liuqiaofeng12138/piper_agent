"""市场竞品调研：搜索与网页抓取。"""

from __future__ import annotations

import re

import httpx


def web_search(query: str, max_results: int = 5) -> str:
    q = (query or "").strip()
    if not q:
        return "（搜索关键词为空）"
    try:
        from duckduckgo_search import DDGS
    except ImportError as e:
        raise RuntimeError("请安装 duckduckgo-search") from e

    items: list[dict] = []
    with DDGS() as ddgs:
        for r in ddgs.text(q, max_results=max_results):
            items.append(r)
    if not items:
        return f"未检索到与「{q}」相关的公开结果（可换关键词重试）"

    lines: list[str] = [f"关键词: {q}", ""]
    for i, it in enumerate(items, 1):
        title = it.get("title") or "（无标题）"
        href = it.get("href") or it.get("link") or ""
        body = (it.get("body") or it.get("snippet") or "").strip()
        lines.append(f"{i}. **{title}**\n   - 链接: {href}\n   - 摘要: {body}")
    return "\n".join(lines)


def scrape_webpage(url: str, timeout_seconds: float = 25.0, max_chars: int = 12000) -> str:
    u = (url or "").strip()
    if not u.startswith(("http://", "https://")):
        raise ValueError("url 必须以 http:// 或 https:// 开头")
    headers = {
        "User-Agent": (
            "Mozilla/5.0 (compatible; PiperEnterpriseOps/1.0; +https://github.com/piper-agent)"
        ),
    }
    with httpx.Client(timeout=timeout_seconds, follow_redirects=True, headers=headers) as client:
        resp = client.get(u)
        resp.raise_for_status()
        raw = resp.text

    text = re.sub(r"<script[\s\S]*?</script>", " ", raw, flags=re.IGNORECASE)
    text = re.sub(r"<style[\s\S]*?</style>", " ", text, flags=re.IGNORECASE)
    text = re.sub(r"<[^>]+>", " ", text)
    text = re.sub(r"\s+", " ", text).strip()
    if len(text) > max_chars:
        text = text[:max_chars] + " …（已截断）"
    return f"来源: {u}\n\n{text}"
