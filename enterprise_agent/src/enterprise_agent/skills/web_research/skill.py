from __future__ import annotations

from typing import Any

from enterprise_agent.config.loader import AgentConfig
from enterprise_agent.skills.base import Skill, TaskContext
from enterprise_agent.skills.loader import load_tools_schema
from enterprise_agent.skills.manifest import SkillManifest
from enterprise_agent.skills.manifest_skill import ManifestSkill
from enterprise_agent.tools.web import scrape_webpage, web_search


class WebResearchSkill(ManifestSkill):
    def __init__(
        self,
        manifest: SkillManifest,
        max_search_results: int,
        scrape_timeout: float,
        scrape_max_chars: int,
    ) -> None:
        super().__init__(manifest, load_tools_schema(manifest.skill_dir))
        self._max_search = max_search_results
        self._scrape_timeout = scrape_timeout
        self._scrape_max_chars = scrape_max_chars

    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        if name == "web_search":
            q = str(arguments.get("query") or "")
            n = int(arguments.get("max_results") or self._max_search)
            return web_search(q, max_results=min(n, self._max_search))
        if name == "scrape_webpage":
            return scrape_webpage(
                str(arguments.get("url") or ""),
                timeout_seconds=self._scrape_timeout,
                max_chars=self._scrape_max_chars,
            )
        return f"未知工具: {name}"


def create_skill(manifest: SkillManifest, cfg: AgentConfig) -> Skill:
    ent = cfg.enterprise_ops
    return WebResearchSkill(
        manifest,
        max_search_results=ent.web_search_max_results,
        scrape_timeout=ent.scrape_timeout_seconds,
        scrape_max_chars=ent.scrape_max_chars,
    )
