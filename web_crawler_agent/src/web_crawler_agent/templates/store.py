"""校验通过的 Piper 模版持久化（Runtime meta + 本地 JSON 归档）。"""

from __future__ import annotations

import json
import re
import time
from pathlib import Path
from typing import Any

from web_crawler_agent.clients.runtime_client import RuntimeClient
from web_crawler_agent.config.loader import AgentConfig
from web_crawler_agent.harness.session import SessionState
from web_crawler_agent.rag.template_index import TemplateIndex


def resolve_saved_templates_dir(cfg: AgentConfig) -> Path | None:
    raw = cfg.harness.saved_templates_dir
    if raw is None:
        if cfg.config_path:
            return (cfg.config_path.parent / ".." / ".." / "data" / "saved_templates").resolve()
        return None
    path = Path(raw)
    if not path.is_absolute() and cfg.config_path:
        path = (cfg.config_path.parent / path).resolve()
    return path


def _safe_template_filename(template_id: str) -> str:
    tid = (template_id or "template").strip()
    tid = re.sub(r'[<>:"/\\|?*\s]+', "_", tid)
    tid = tid.strip("._") or "template"
    return tid[:120]


def save_template_file(cfg: AgentConfig, template_id: str, doc: dict[str, Any]) -> Path | None:
    base = resolve_saved_templates_dir(cfg)
    if base is None:
        return None
    base.mkdir(parents=True, exist_ok=True)
    fname = _safe_template_filename(template_id) + ".json"
    path = base / fname
    body = json.dumps(doc, ensure_ascii=False, indent=2)
    path.write_text(body + "\n", encoding="utf-8")
    manifest = base / "_manifest.jsonl"
    row = {
        "ts": time.time(),
        "template_id": template_id,
        "file": fname,
        "name": doc.get("name"),
        "domain": doc.get("domain"),
    }
    with manifest.open("a", encoding="utf-8") as f:
        f.write(json.dumps(row, ensure_ascii=False) + "\n")
    return path


def persist_validated_template(
    *,
    client: RuntimeClient,
    index: TemplateIndex,
    session: SessionState,
    cfg: AgentConfig,
    doc: dict[str, Any],
    template_id: str,
) -> dict[str, Any]:
    """Upsert 到 Runtime，并写入本地 saved_templates 目录。"""
    tid = (template_id or str(doc.get("id") or doc.get("name") or "")).strip()
    if not tid:
        tid = f"tpl_{int(time.time())}"
        doc = {**doc, "id": tid}

    payload = json.dumps(doc, ensure_ascii=False).encode("utf-8")
    resp = client.upsert_template(
        template_id=tid,
        name=str(doc.get("name") or tid),
        json_payload=payload,
        session_id=session.session_id,
    )
    saved_id = resp.template_id or tid
    session.validated_template_ids.add(saved_id)
    session.last_template_id = saved_id
    session.pending_template = doc

    out: dict[str, Any] = {
        "template_id": saved_id,
        "saved_to_meta": True,
        "saved_to_file": None,
    }
    if cfg.harness.auto_save_templates:
        path = save_template_file(cfg, saved_id, doc)
        out["saved_to_file"] = str(path) if path else None
        index.refresh()
    return out
