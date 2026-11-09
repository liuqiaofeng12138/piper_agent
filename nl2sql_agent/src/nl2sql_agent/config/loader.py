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
class NL2SQLConfig:
    dsn: str = "postgresql://piper:piper_nl2sql@127.0.0.1:5433/semi_dw"
    use_llm_extract: bool = False
    use_llm_explain: bool = True
    catalog_path: str | None = None


@dataclass
class AgentConfig:
    llm: LLMConfig = field(default_factory=LLMConfig)
    nl2sql: NL2SQLConfig = field(default_factory=NL2SQLConfig)
    config_path: Path | None = None


def default_config_path() -> Path:
    return Path(__file__).resolve().parents[4] / "deploy" / "config" / "local.yaml"


def load_agent_config(path: str | Path | None = None) -> AgentConfig:
    cfg = AgentConfig()
    config_file = Path(path) if path else default_config_path()
    cfg.config_path = config_file
    if config_file.is_file():
        raw: dict[str, Any] = yaml.safe_load(config_file.read_text(encoding="utf-8")) or {}
        _apply_raw(cfg, raw, config_file)
    elif path:
        raise FileNotFoundError(f"config not found: {config_file}")
    return cfg


def _apply_raw(cfg: AgentConfig, raw: dict[str, Any], config_file: Path) -> None:
    llm = raw.get("llm") or {}
    base = llm.get("base_url")
    cfg.llm = LLMConfig(
        provider=str(llm.get("provider", cfg.llm.provider)),
        model=str(llm.get("model", cfg.llm.model)),
        base_url=str(base).strip() if base else None,
        api_key=str(llm.get("api_key") or "").strip(),
    )
    ns = raw.get("nl2sql") or {}
    dsn = str(ns.get("dsn") or cfg.nl2sql.dsn)
    cfg.nl2sql = NL2SQLConfig(
        dsn=dsn,
        use_llm_extract=bool(ns.get("use_llm_extract", cfg.nl2sql.use_llm_extract)),
        use_llm_explain=bool(ns.get("use_llm_explain", cfg.nl2sql.use_llm_explain)),
        catalog_path=str(ns["catalog_path"]) if ns.get("catalog_path") else None,
    )
    if not ns.get("dsn"):
        cfg.nl2sql.dsn = _default_dsn_from_example(config_file)


def _default_dsn_from_example(config_file: Path) -> str:
    return "postgresql://piper:piper_nl2sql@127.0.0.1:5433/semi_dw"


def worker_listen_address(path: str | Path | None = None, agent_id: str = "nl2sql") -> str:
    fallback = "127.0.0.1:15067"
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
