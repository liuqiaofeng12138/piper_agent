from __future__ import annotations

from typing import Any

from piper_agent.harness.audit import audit_log
from piper_agent.tools.handlers import ToolHandlers
from piper_agent.tools.schemas import TOOL_SCHEMAS


class ToolRegistry:
    def __init__(self, handlers: ToolHandlers) -> None:
        self.handlers = handlers
        self.session = handlers.session
        self.config = handlers.config

    def openai_tools(self) -> list[dict[str, Any]]:
        return list(TOOL_SCHEMAS)

    def execute(self, name: str, arguments: str | dict[str, Any]) -> str:
        if isinstance(arguments, str):
            import json

            args = json.loads(arguments) if arguments.strip() else {}
        else:
            args = arguments
        result = self.handlers.dispatch(name, args)
        audit_log(
            self.config.harness.audit_log,
            self.session.session_id,
            "tool",
            {"tool": name, "args": args, "result_preview": result[:500]},
        )
        return result
