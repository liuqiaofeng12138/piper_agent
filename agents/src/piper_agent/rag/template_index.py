from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from piper_agent.clients.runtime_client import RuntimeClient


@dataclass
class TemplateHit:
    template_id: str
    name: str
    domain: str
    snippet: str
    score: float
    source: str
    template_json: str | None = None


@dataclass
class TemplateIndex:
    client: RuntimeClient
    examples_dir: Path | None = None
    _entries: list[dict[str, Any]] = field(default_factory=list)

    def refresh(self, *, list_size: int = 200) -> int:
        self._entries.clear()
        try:
            resp = self.client.list_templates(size=list_size)
            for t in resp.templates:
                payload = t.json_payload.decode("utf-8", errors="replace") if t.json_payload else ""
                self._entries.append(
                    {
                        "id": t.id or t.name,
                        "name": t.name,
                        "domain": t.domain,
                        "text": f"{t.name} {t.domain} {payload[:4000]}",
                        "source": "meta",
                    }
                )
        except Exception:
            pass
        if self.examples_dir and self.examples_dir.is_dir():
            for fp in self.examples_dir.glob("*.json"):
                try:
                    doc = json.loads(fp.read_text(encoding="utf-8"))
                    full = json.dumps(doc, ensure_ascii=False)
                    self._entries.append(
                        {
                            "id": doc.get("id") or fp.stem,
                            "name": doc.get("name", fp.stem),
                            "domain": doc.get("domain", ""),
                            "text": full[:6000],
                            "template_json": full,
                            "source": f"example:{fp.name}",
                            "example_file": fp.name,
                        }
                    )
                except (json.JSONDecodeError, OSError):
                    continue
        return len(self._entries)

    def search(self, query: str, *, top_k: int = 5, prefer_shared: bool = False) -> list[TemplateHit]:
        if not self._entries:
            self.refresh()
        q = query.lower().strip()
        if not q:
            return []
        tokens = [t for t in re.split(r"\W+", q) if t]
        wants_shared = prefer_shared or any(
            k in q for k in ("shared", "example", "示例", "examples/templates")
        )
        hits: list[TemplateHit] = []
        for e in self._entries:
            text = e["text"].lower()
            score = sum(1.0 for t in tokens if t in text)
            if e["name"] and e["name"].lower() in q:
                score += 2.0
            if e["domain"] and e["domain"].lower() in q:
                score += 1.5
            source = str(e.get("source") or "meta")
            if source.startswith("example:"):
                if wants_shared:
                    score += 8.0
            elif wants_shared:
                score -= 2.0
            if score <= 0:
                continue
            hits.append(
                TemplateHit(
                    template_id=str(e["id"]),
                    name=str(e.get("name") or ""),
                    domain=str(e.get("domain") or ""),
                    snippet=e["text"][:500],
                    score=score,
                    source=source,
                    template_json=e.get("template_json"),
                )
            )
        hits.sort(key=lambda h: (-h.score, 0 if h.source.startswith("example:") else 1))
        # When user asks for shared, drop duplicate meta row if example exists for same id.
        if wants_shared:
            seen_example: set[str] = set()
            filtered: list[TemplateHit] = []
            for h in hits:
                if h.source.startswith("example:"):
                    seen_example.add(h.template_id)
            for h in hits:
                if h.source == "meta" and h.template_id in seen_example:
                    continue
                filtered.append(h)
            hits = filtered
        return hits[:top_k]

    def load_shared_example(self, key: str) -> tuple[dict[str, Any], str] | None:
        """Load exact template JSON from shared/examples (by id, stem, or filename)."""
        if not self.examples_dir or not self.examples_dir.is_dir():
            return None
        key = key.strip()
        if not key:
            return None
        candidates = [key]
        if not key.endswith(".json"):
            candidates.append(f"{key}.json")
        for name in candidates:
            fp = self.examples_dir / name
            if fp.is_file():
                doc = json.loads(fp.read_text(encoding="utf-8"))
                return doc, fp.name
        key_l = key.lower()
        for fp in self.examples_dir.glob("*.json"):
            try:
                doc = json.loads(fp.read_text(encoding="utf-8"))
            except (json.JSONDecodeError, OSError):
                continue
            tid = str(doc.get("id") or fp.stem)
            if tid.lower() == key_l or fp.stem.lower() == key_l or doc.get("name", "").lower() == key_l:
                return doc, fp.name
        return None
