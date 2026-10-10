"""HTTP 探站：拉取目标 URL，分析内容类型与结构，供 LLM 写模版时结合示例使用。"""

from __future__ import annotations

import json
import re
import urllib.error
import urllib.request
from typing import Any, Callable
from urllib.parse import urljoin, urlparse

DEFAULT_USER_AGENT = "piper-web-crawler-probe/0.1 (+https://github.com/local-dev)"
def normalize_probe_url(url: str) -> str:
    u = (url or "").strip()
    if not u:
        return ""
    if not u.startswith(("http://", "https://")):
        u = "https://" + u
    return u


def probe_url(
    url: str,
    *,
    max_body_bytes: int = 65536,
    timeout_seconds: float = 25.0,
    fetch: Callable[[str, float, int], tuple[int, str, dict[str, str], bytes]] | None = None,
) -> dict[str, Any]:
    """探测 URL，返回面向模版编写的结构化摘要（不执行 Piper Run）。"""
    start_url = normalize_probe_url(url)
    if not start_url:
        return {"ok": False, "error": "url is required"}

    max_body_bytes = max(1024, min(int(max_body_bytes), 512 * 1024))

    def _default_fetch(
        target: str, timeout: float, limit: int
    ) -> tuple[int, str, dict[str, str], bytes]:
        req = urllib.request.Request(
            target,
            headers={"User-Agent": DEFAULT_USER_AGENT, "Accept": "*/*"},
            method="GET",
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            status = int(getattr(resp, "status", 200) or 200)
            final = resp.geturl() or target
            headers = {k.lower(): v for k, v in resp.headers.items()}
            raw = resp.read(limit + 1)
            if len(raw) > limit:
                raw = raw[:limit]
            return status, final, headers, raw

    fetch_fn = fetch or _default_fetch
    redirect_chain: list[str] = [start_url]
    current = start_url
    status = 0
    headers: dict[str, str] = {}
    body = b""

    try:
        status, current, headers, body = fetch_fn(start_url, timeout_seconds, max_body_bytes)
        if current != start_url:
            redirect_chain.append(current)
    except urllib.error.HTTPError as e:
        status = e.code
        current = e.url or start_url
        if current != start_url:
            redirect_chain.append(current)
        headers = {k.lower(): v for k, v in (e.headers.items() if e.headers else [])}
        body = e.read(max_body_bytes) if e.fp else b""
    except Exception as e:  # noqa: BLE001
        return {"ok": False, "error": str(e), "url": start_url}

    content_type = (headers.get("content-type") or "").split(";")[0].strip().lower()
    charset = _guess_charset(headers.get("content-type", ""), body)
    text = body.decode(charset, errors="replace")
    truncated = len(body) >= max_body_bytes

    out: dict[str, Any] = {
        "ok": True,
        "url": start_url,
        "final_url": current,
        "status_code": status,
        "content_type": content_type or "unknown",
        "charset": charset,
        "body_bytes_read": len(body),
        "body_truncated": truncated,
        "redirect_chain": redirect_chain,
        "suggested_engine": "http",
        "template_hints": [],
    }

    if "json" in content_type or _looks_like_json(text):
        out.update(_analyze_json(text))
        out["suggested_engine"] = "http"
        out["template_hints"].append(
            "响应为 JSON：优先参考 shared/examples/templates/http_jsonpath.json，"
            "用 JSONPath 在 procedures.fields 中映射字段。"
        )
    elif "html" in content_type or _looks_like_html(text):
        out.update(_analyze_html(text, base_url=current))
        if out.get("likely_spa"):
            out["suggested_engine"] = "chrome"
            out["template_hints"].append(
                "页面疑似 SPA/强 JS：Http 可能拿不到列表数据，建议 builder.type=Chrome，"
                "参考 chrome_navigate_stub，并配合 Regex/Selector 或抓包接口。"
            )
        else:
            out["template_hints"].append(
                "响应为 HTML：可参考 http_regex_index.json 用 Regex 抽字段，"
                "或确认是否有独立 JSON API 再改用 JSONPath。"
            )
    else:
        out["body_preview"] = _preview_text(text, 4000)
        out["template_hints"].append("非 JSON/HTML，请确认 URL 是否为接口或需登录。")

    out["body_preview"] = out.get("body_preview") or _preview_text(text, 4000)
    return out


def _guess_charset(content_type: str, body: bytes) -> str:
    m = re.search(r"charset=([^\s;]+)", content_type, re.I)
    if m:
        return m.group(1).strip("\"'") or "utf-8"
    m = re.search(rb"charset=[\"']?([a-zA-Z0-9_-]+)", body[:2048], re.I)
    if m:
        return m.group(1).decode("ascii", errors="ignore") or "utf-8"
    return "utf-8"


def _looks_like_json(text: str) -> bool:
    t = text.lstrip()
    return t.startswith("{") or t.startswith("[")


def _looks_like_html(text: str) -> bool:
    t = text.lstrip().lower()
    return t.startswith("<!doctype") or t.startswith("<html") or "<body" in t[:2000].lower()


def _analyze_json(text: str) -> dict[str, Any]:
    hints: dict[str, Any] = {"format": "json", "json_parse_ok": False}
    try:
        data = json.loads(text)
    except json.JSONDecodeError as e:
        hints["json_error"] = str(e)
        hints["body_preview"] = _preview_text(text, 4000)
        return hints

    hints["json_parse_ok"] = True
    hints["json_top_level_type"] = type(data).__name__
    if isinstance(data, dict):
        hints["json_top_level_keys"] = list(data.keys())[:40]
        hints["json_sample_paths"] = _json_path_samples(data, max_paths=12)
    elif isinstance(data, list):
        hints["json_array_length"] = len(data)
        if data and isinstance(data[0], dict):
            hints["json_first_item_keys"] = list(data[0].keys())[:30]
            hints["json_sample_paths"] = _json_path_samples(data[0], max_paths=12, prefix="$[0]")
    hints["body_preview"] = _preview_text(text, 6000)
    return hints


def _json_path_samples(obj: Any, max_paths: int, prefix: str = "$") -> list[str]:
    paths: list[str] = []

    def walk(node: Any, path: str, depth: int) -> None:
        if len(paths) >= max_paths or depth > 4:
            return
        if isinstance(node, dict):
            for k, v in list(node.items())[:8]:
                p = f"{path}.{k}" if path != "$" else f"$.{k}"
                paths.append(p)
                walk(v, p, depth + 1)
        elif isinstance(node, list) and node:
            walk(node[0], f"{path}[0]", depth + 1)

    walk(obj, prefix, 0)
    return paths[:max_paths]


def _analyze_html(text: str, base_url: str) -> dict[str, Any]:
    title_m = re.search(r"<title[^>]*>([^<]+)</title>", text, re.I | re.S)
    title = re.sub(r"\s+", " ", title_m.group(1)).strip() if title_m else ""
    desc_m = re.search(
        r'<meta[^>]+name=["\']description["\'][^>]+content=["\']([^"\']+)',
        text,
        re.I,
    )
    description = desc_m.group(1).strip() if desc_m else ""

    script_count = len(re.findall(r"<script\b", text, re.I))
    text_len = len(re.sub(r"<[^>]+>", " ", text))
    likely_spa = script_count >= 8 and text_len < 8000

    api_links = _extract_api_like_links(text, base_url)

    return {
        "format": "html",
        "title": title,
        "meta_description": description,
        "script_tag_count": script_count,
        "visible_text_chars": text_len,
        "likely_spa": likely_spa,
        "api_like_links": api_links[:15],
        "body_preview": _preview_text(_strip_tags(text), 5000),
    }


def _extract_api_like_links(html: str, base_url: str) -> list[str]:
    hosts = {urlparse(base_url).netloc}
    found: list[str] = []
    for m in re.finditer(r"""href=["']([^"']+)["']""", html, re.I):
        href = m.group(1).strip()
        if not href or href.startswith(("#", "javascript:", "mailto:")):
            continue
        full = urljoin(base_url, href)
        p = urlparse(full)
        if p.netloc and p.netloc not in hosts:
            continue
        lower = full.lower()
        if any(x in lower for x in ("/api/", ".json", "/ajax/", "graphql")):
            found.append(full)
    return found


def _strip_tags(html: str) -> str:
    t = re.sub(r"<(script|style)[^>]*>.*?</\1>", " ", html, flags=re.I | re.S)
    t = re.sub(r"<[^>]+>", " ", t)
    return re.sub(r"\s+", " ", t).strip()


def _preview_text(text: str, limit: int) -> str:
    t = re.sub(r"\s+", " ", (text or "").strip())
    if len(t) <= limit:
        return t
    return t[: limit - 1] + "…"
