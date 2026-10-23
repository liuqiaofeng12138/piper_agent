from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class SkillManifest:
    """从技能目录 SKILL.md 解析出的标准元数据。"""

    skill_id: str
    name: str
    description: str
    system_prompt: str
    route_keywords: tuple[str, ...]
    skill_dir: Path
    order: int = 100
