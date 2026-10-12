from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml


@dataclass
class Word2PPTConfig:
    artifacts_dir: str = "../../data/artifacts"
    max_slides_per_doc: int = 24
    max_bullets_per_slide: int = 6
    max_chars_per_bullet: int = 220


@dataclass
class AgentConfig:
    word2ppt: Word2PPTConfig = field(default_factory=Word2PPTConfig)
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


def resolve_artifacts_dir(cfg: AgentConfig) -> Path:
    base = cfg.config_path.parent if cfg.config_path else default_config_path().parent
    p = Path(cfg.word2ppt.artifacts_dir)
    if not p.is_absolute():
        p = (base / p).resolve()
    p.mkdir(parents=True, exist_ok=True)
    return p


def _apply_raw(cfg: AgentConfig, raw: dict[str, Any], config_file: Path) -> None:
    w2p = raw.get("word2ppt") or {}
    artifacts = raw.get("artifacts") or {}
    art_dir = w2p.get("artifacts_dir") or artifacts.get("dir")
    cfg.word2ppt = Word2PPTConfig(
        artifacts_dir=str(art_dir or cfg.word2ppt.artifacts_dir),
        max_slides_per_doc=int(w2p.get("max_slides_per_doc", cfg.word2ppt.max_slides_per_doc)),
        max_bullets_per_slide=int(
            w2p.get("max_bullets_per_slide", cfg.word2ppt.max_bullets_per_slide)
        ),
        max_chars_per_bullet=int(
            w2p.get("max_chars_per_bullet", cfg.word2ppt.max_chars_per_bullet)
        ),
    )
    _ = config_file


def worker_listen_address(path: str | Path | None = None, agent_id: str = "word2ppt") -> str:
    fallback = "127.0.0.1:15065"
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
