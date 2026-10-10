from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml


@dataclass
class LLMConfig:
    provider: str = "openai"
    model: str = "gpt-4o"
    base_url: str | None = None
    api_key: str = ""


@dataclass
class RAGConfig:
    chunk_size: int = 900
    chunk_overlap: int = 120
    top_k: int = 8
    max_history_messages: int = 24


@dataclass
class AgentConfig:
    llm: LLMConfig = field(default_factory=LLMConfig)
    rag: RAGConfig = field(default_factory=RAGConfig)
    config_path: Path | None = None


def default_config_path() -> Path:
    return Path(__file__).resolve().parents[4] / "deploy" / "config" / "local.yaml"


def load_agent_config(path: str | Path | None = None) -> AgentConfig:
    cfg = AgentConfig()
    config_file = Path(path) if path else default_config_path()
    cfg.config_path = config_file
    if config_file.is_file():
        raw: dict[str, Any] = yaml.safe_load(config_file.read_text(encoding="utf-8")) or {}
        _apply_raw(cfg, raw)
    elif path:
        raise FileNotFoundError(f"config not found: {config_file}")
    return cfg


def _apply_raw(cfg: AgentConfig, raw: dict[str, Any]) -> None:
    llm = raw.get("llm") or {}
    base = llm.get("base_url")
    cfg.llm = LLMConfig(
        provider=str(llm.get("provider", cfg.llm.provider)),
        model=str(llm.get("model", cfg.llm.model)),
        base_url=str(base).strip() if base else None,
        api_key=str(llm.get("api_key") or "").strip(),
    )
    rag = raw.get("rag") or {}
    cfg.rag = RAGConfig(
        chunk_size=int(rag.get("chunk_size", cfg.rag.chunk_size)),
        chunk_overlap=int(rag.get("chunk_overlap", cfg.rag.chunk_overlap)),
        top_k=int(rag.get("top_k", cfg.rag.top_k)),
        max_history_messages=int(rag.get("max_history_messages", cfg.rag.max_history_messages)),
    )


def worker_listen_address(path: str | Path | None = None, agent_id: str = "doc_rag") -> str:
    fallback = "127.0.0.1:15063"
    config_file = Path(path) if path else default_config_path()
    if not config_file.is_file():
        return fallback
    raw: dict[str, Any] = yaml.safe_load(config_file.read_text(encoding="utf-8")) or {}
    agents = raw.get("agents") or {}
    entry: dict[str, Any] = {}
    if isinstance(agents, list):
        for item in agents:
            if isinstance(item, dict) and item.get("id") == agent_id:
                entry = item
                break
    elif isinstance(agents, dict):
        entry = agents.get(agent_id) or {}
    listen = entry.get("listen") or entry.get("address")
    return str(listen or fallback)
