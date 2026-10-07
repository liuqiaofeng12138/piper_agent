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
class HarnessConfig:
    max_steps: int = 20
    require_validate_before_run: bool = True
    max_runs_per_session: int = 10
    audit_log: Path | None = None


@dataclass
class AgentConfig:
    runtime_address: str = "127.0.0.1:50051"
    llm: LLMConfig = field(default_factory=LLMConfig)
    harness: HarnessConfig = field(default_factory=HarnessConfig)
    examples_dir: Path | None = None
    prompts_dir: Path | None = None
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

    repo = _find_repo_root()
    if repo:
        cfg.examples_dir = repo / "shared" / "examples" / "templates"
        cfg.prompts_dir = repo / "shared" / "prompts"
    return cfg


def _apply_raw(cfg: AgentConfig, raw: dict[str, Any]) -> None:
    rt = raw.get("runtime") or {}
    cfg.runtime_address = str(rt.get("address", cfg.runtime_address))
    llm = raw.get("llm") or {}
    base = llm.get("base_url")
    cfg.llm = LLMConfig(
        provider=str(llm.get("provider", cfg.llm.provider)),
        model=str(llm.get("model", cfg.llm.model)),
        base_url=str(base).strip() if base else None,
        api_key=str(llm.get("api_key") or "").strip(),
    )
    har = raw.get("harness") or {}
    audit_raw = har.get("audit_log")
    audit_path = Path(audit_raw) if audit_raw else None
    if audit_path and not audit_path.is_absolute() and cfg.config_path:
        audit_path = (cfg.config_path.parent / audit_path).resolve()
    cfg.harness = HarnessConfig(
        max_steps=int(har.get("max_steps", cfg.harness.max_steps)),
        require_validate_before_run=bool(
            har.get("require_validate_before_run", cfg.harness.require_validate_before_run)
        ),
        max_runs_per_session=int(har.get("max_runs_per_session", cfg.harness.max_runs_per_session)),
        audit_log=audit_path,
    )


def worker_listen_address(path: str | Path | None = None) -> str:
    config_file = Path(path) if path else default_config_path()
    if not config_file.is_file():
        return "127.0.0.1:15061"
    raw: dict[str, Any] = yaml.safe_load(config_file.read_text(encoding="utf-8")) or {}
    agents = raw.get("agents") or {}
    wc = agents.get("web_crawler") or {}
    listen = wc.get("listen") or wc.get("address")
    return str(listen or "127.0.0.1:15061")


def _find_repo_root() -> Path | None:
    here = Path(__file__).resolve()
    for p in here.parents:
        if (p / "shared" / "examples").is_dir() and (p / "piper_agent").is_dir():
            return p / "piper_agent"
        if p.name == "piper_agent" and (p / "shared").is_dir():
            return p
    return None
