from __future__ import annotations

from typing import Any

from enterprise_agent.skills.base import Skill, SkillDefinition, TaskContext
from enterprise_agent.skills.manifest import SkillManifest


class ManifestSkill(Skill):
    """基于 SKILL.md + tools.json 的 Skill 基类，子类仅实现 execute_tool。"""

    def __init__(self, manifest: SkillManifest, tools: list[dict[str, Any]]) -> None:
        self.manifest = manifest
        self._tools = tools
        self.definition = SkillDefinition(
            skill_id=manifest.skill_id,
            display_name=manifest.name,
            description=manifest.description,
            system_prompt=manifest.system_prompt,
        )

    def openai_tools(self) -> list[dict[str, Any]]:
        return list(self._tools)

    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        raise NotImplementedError
