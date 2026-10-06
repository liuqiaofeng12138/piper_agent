from __future__ import annotations

import json
from typing import Any

from piper_agent.config.loader import AgentConfig
from piper_agent.harness.registry import ToolRegistry
from piper_agent.harness.session import SessionState


class AgentLoop:
    def __init__(
        self,
        registry: ToolRegistry,
        session: SessionState,
        config: AgentConfig,
        *,
        llm_client: Any,
    ) -> None:
        self.registry = registry
        self.session = session
        self.config = config
        self.llm = llm_client

    def run_turn(self, user_text: str) -> str:
        self.session.messages.append({"role": "user", "content": user_text})
        system = self._system_prompt()
        for step in range(self.config.harness.max_steps):
            response = self.llm.chat.completions.create(
                model=self.config.llm.model,
                messages=[{"role": "system", "content": system}, *self.session.messages],
                tools=self.registry.openai_tools(),
                tool_choice="auto",
            )
            msg = response.choices[0].message
            assistant_record: dict[str, Any] = {
                "role": "assistant",
                "content": msg.content or "",
            }
            if msg.tool_calls:
                assistant_record["tool_calls"] = [
                    {
                        "id": tc.id,
                        "type": "function",
                        "function": {"name": tc.function.name, "arguments": tc.function.arguments},
                    }
                    for tc in msg.tool_calls
                ]
            self.session.messages.append(assistant_record)

            if not msg.tool_calls:
                return (msg.content or "").strip() or "(empty response)"

            for tc in msg.tool_calls:
                result = self.registry.execute(tc.function.name, tc.function.arguments or "{}")
                self.session.messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tc.id,
                        "content": result,
                    }
                )
        return "已达到最大步数，请缩小任务或继续对话。"

    def _system_prompt(self) -> str:
        path = self.config.prompts_dir / "orchestrator_system.md" if self.config.prompts_dir else None
        if path and path.is_file():
            return path.read_text(encoding="utf-8")
        return "You are Piper Agent. Use tools to validate templates before run."
