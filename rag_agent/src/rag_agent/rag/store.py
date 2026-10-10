from __future__ import annotations

import re
import threading
from dataclasses import dataclass


def _chunk_text(text: str, chunk_size: int, overlap: int) -> list[str]:
    text = re.sub(r"\s+", " ", text).strip()
    if not text:
        return []
    if len(text) <= chunk_size:
        return [text]
    chunks: list[str] = []
    start = 0
    while start < len(text):
        end = min(len(text), start + chunk_size)
        chunks.append(text[start:end])
        if end >= len(text):
            break
        start = max(0, end - overlap)
    return chunks


@dataclass
class _DocRecord:
    filename: str
    chunks: list[str]


class DocumentStore:
    """按会话保存文档分块（进程内；重启后需重新上传）。"""

    def __init__(self, chunk_size: int = 900, chunk_overlap: int = 120) -> None:
        self._chunk_size = chunk_size
        self._chunk_overlap = chunk_overlap
        self._by_conv: dict[str, list[_DocRecord]] = {}
        self._lock = threading.Lock()

    def add_document(self, conversation_id: str, filename: str, text: str) -> int:
        pieces = _chunk_text(text, self._chunk_size, self._chunk_overlap)
        if not pieces:
            return 0
        with self._lock:
            records = self._by_conv.setdefault(conversation_id, [])
            records.append(_DocRecord(filename=filename, chunks=pieces))
        return len(pieces)

    def list_filenames(self, conversation_id: str) -> list[str]:
        with self._lock:
            return [r.filename for r in self._by_conv.get(conversation_id, [])]

    def retrieve(self, conversation_id: str, query: str, top_k: int = 8) -> list[str]:
        with self._lock:
            records = self._by_conv.get(conversation_id, [])
            all_chunks: list[str] = []
            for rec in records:
                for c in rec.chunks:
                    all_chunks.append(f"[{rec.filename}] {c}")

        if not all_chunks:
            return []

        terms = {t for t in re.split(r"\W+", query.lower()) if len(t) >= 2}
        scored: list[tuple[int, str]] = []
        for chunk in all_chunks:
            lower = chunk.lower()
            score = sum(1 for t in terms if t in lower)
            scored.append((score, chunk))
        scored.sort(key=lambda x: (-x[0], x[1]))
        if scored[0][0] > 0:
            return [c for _, c in scored[:top_k]]
        return [c for _, c in scored[:top_k]]
