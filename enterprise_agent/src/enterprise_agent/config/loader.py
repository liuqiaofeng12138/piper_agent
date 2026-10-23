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
class EnterpriseOpsConfig:
    sql_db_path: str = "../../data/enterprise_demo.sqlite"
    max_skill_steps: int = 8
    max_history_messages: int = 20
    web_search_max_results: int = 5
    scrape_max_chars: int = 12000
    scrape_timeout_seconds: float = 25.0


@dataclass
class AgentConfig:
    llm: LLMConfig = field(default_factory=LLMConfig)
    enterprise_ops: EnterpriseOpsConfig = field(default_factory=EnterpriseOpsConfig)
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


def resolve_sql_db_path(cfg: AgentConfig) -> Path:
    base = cfg.config_path.parent if cfg.config_path else default_config_path().parent
    p = Path(cfg.enterprise_ops.sql_db_path)
    if not p.is_absolute():
        p = (base / p).resolve()
    p.parent.mkdir(parents=True, exist_ok=True)
    return p


def _apply_raw(cfg: AgentConfig, raw: dict[str, Any]) -> None:
    llm = raw.get("llm") or {}
    base = llm.get("base_url")
    cfg.llm = LLMConfig(
        provider=str(llm.get("provider", cfg.llm.provider)),
        model=str(llm.get("model", cfg.llm.model)),
        base_url=str(base).strip() if base else None,
        api_key=str(llm.get("api_key") or "").strip(),
    )
    ent = raw.get("enterprise_ops") or {}
    cfg.enterprise_ops = EnterpriseOpsConfig(
        sql_db_path=str(ent.get("sql_db_path", cfg.enterprise_ops.sql_db_path)),
        max_skill_steps=int(ent.get("max_skill_steps", cfg.enterprise_ops.max_skill_steps)),
        max_history_messages=int(
            ent.get("max_history_messages", cfg.enterprise_ops.max_history_messages)
        ),
        web_search_max_results=int(
            ent.get("web_search_max_results", cfg.enterprise_ops.web_search_max_results)
        ),
        scrape_max_chars=int(ent.get("scrape_max_chars", cfg.enterprise_ops.scrape_max_chars)),
        scrape_timeout_seconds=float(
            ent.get("scrape_timeout_seconds", cfg.enterprise_ops.scrape_timeout_seconds)
        ),
    )


def worker_listen_address(path: str | Path | None = None, agent_id: str = "enterprise_ops") -> str:
    fallback = "127.0.0.1:15066"
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
