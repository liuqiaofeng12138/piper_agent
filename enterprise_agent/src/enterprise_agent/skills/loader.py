from __future__ import annotations

import importlib
import json
import re
from pathlib import Path
from typing import Any

import yaml

from enterprise_agent.config.loader import AgentConfig, resolve_sql_db_path
from enterprise_agent.skills.base import Skill
from enterprise_agent.skills.manifest import SkillManifest

_SKILLS_ROOT = Path(__file__).resolve().parent
_FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n(.*)$", re.DOTALL | re.MULTILINE)


def skills_root() -> Path:
    return _SKILLS_ROOT


def discover_skill_dirs() -> list[Path]:
    """扫描含 SKILL.md 的子目录（标准 Skill 包布局）。"""
    dirs = [p for p in _SKILLS_ROOT.iterdir() if p.is_dir() and (p / "SKILL.md").is_file()]
    manifests = [load_manifest(p) for p in dirs]
    pairs = sorted(zip(dirs, manifests, strict=True), key=lambda x: (x[1].order, x[1].skill_id))
    return [p for p, _ in pairs]


def load_manifest(skill_dir: Path) -> SkillManifest:
    path = skill_dir / "SKILL.md"
    text = path.read_text(encoding="utf-8")
    m = _FRONTMATTER_RE.match(text)
    if not m:
        raise ValueError(f"SKILL.md 缺少 YAML frontmatter: {path}")
    meta: dict[str, Any] = yaml.safe_load(m.group(1)) or {}
    body = m.group(2).strip()

    skill_id = str(meta.get("id") or skill_dir.name).strip()
    name = str(meta.get("name") or skill_id).strip()
    description = str(meta.get("description") or "").strip()
    keywords = meta.get("route_keywords") or []
    if not isinstance(keywords, list):
        keywords = []
    route_keywords = tuple(str(k).strip() for k in keywords if str(k).strip())
    order = int(meta.get("order", 100))

    return SkillManifest(
        skill_id=skill_id,
        name=name,
        description=description,
        system_prompt=body,
        route_keywords=route_keywords,
        skill_dir=skill_dir,
        order=order,
    )


def load_tools_schema(skill_dir: Path) -> list[dict[str, Any]]:
    path = skill_dir / "tools.json"
    if not path.is_file():
        return []
    data = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(data, list):
        raise ValueError(f"tools.json 须为 OpenAI tools 数组: {path}")
    return data


def load_all_manifests() -> list[SkillManifest]:
    return [load_manifest(d) for d in discover_skill_dirs()]


def build_skill(manifest: SkillManifest, cfg: AgentConfig) -> Skill:
    """从技能目录 skill.py 的 create_skill 工厂构建运行时 Skill 实例。"""
    module_name = f"enterprise_agent.skills.{manifest.skill_dir.name}.skill"
    mod = importlib.import_module(module_name)
    create = getattr(mod, "create_skill", None)
    if create is None:
        raise RuntimeError(f"{module_name} 须定义 create_skill(manifest, cfg) -> Skill")
    return create(manifest, cfg)


def build_registry_skills(cfg: AgentConfig) -> dict[str, Skill]:
    _ = resolve_sql_db_path(cfg)
    out: dict[str, Skill] = {}
    for skill_dir in discover_skill_dirs():
        manifest = load_manifest(skill_dir)
        out[manifest.skill_id] = build_skill(manifest, cfg)
    return out
