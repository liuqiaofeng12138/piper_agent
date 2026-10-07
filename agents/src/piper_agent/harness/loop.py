from __future__ import annotations

import json
from collections.abc import Callable, Iterator
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

    def run_turn_iter(
        self,
        user_text: str,
        *,
        cancel: Callable[[], bool] | None = None,
        delta_chunk_runes: int = 8,
    ) -> Iterator[dict[str, Any]]:
        """流式执行一轮对话，供 Web Worker / gRPC 推送事件。"""
        self.session.messages.append({"role": "user", "content": user_text})
        system = self._system_prompt()
        yield {"type": "run.started", "message": "agent loop started"}

        for step in range(self.config.harness.max_steps):
            if cancel and cancel():
                yield {"type": "run.cancelled", "message": "用户取消"}
                return

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

            if msg.content:
                for piece in _chunk_text((msg.content or "").strip(), delta_chunk_runes):
                    if cancel and cancel():
                        yield {"type": "run.cancelled", "message": "用户取消"}
                        return
                    yield {"type": "message.delta", "delta": piece}

            if not msg.tool_calls:
                final = (msg.content or "").strip() or "(empty response)"
                yield {"type": "message.done", "content": final}
                return

            for tc in msg.tool_calls:
                if cancel and cancel():
                    yield {"type": "run.cancelled", "message": "用户取消"}
                    return
                name = tc.function.name
                args = tc.function.arguments or "{}"
                yield {
                    "type": "tool.call",
                    "tool_name": name,
                    "tool_detail": _truncate(args, 800),
                    "message": "calling",
                }
                result = self.registry.execute(name, args)
                yield {
                    "type": "run.progress",
                    "tool_name": name,
                    "message": _truncate(result, 1200),
                }
                self.session.messages.append(
                    {
                        "role": "tool",
                        "tool_call_id": tc.id,
                        "content": result,
                    }
                )

        final = "已达到最大步数，请缩小任务或继续对话。"
        yield {"type": "message.delta", "delta": final}
        yield {"type": "message.done", "content": final}

    def _system_prompt(self) -> str:
        path = self.config.prompts_dir / "orchestrator_system.md" if self.config.prompts_dir else None
        if path and path.is_file():
            return path.read_text(encoding="utf-8")
        return "You are Piper Agent. Use tools to validate templates before run."


def _chunk_text(text: str, max_runes: int) -> list[str]:
    if not text or max_runes < 1:
        return [text] if text else []
    chunks: list[str] = []
    i = 0
    while i < len(text):
        size = 0
        j = i
        while j < len(text) and size < max_runes:
            j += 1
            size += 1
        chunks.append(text[i:j])
        i = j
    return chunks


def _truncate(s: str, n: int) -> str:
    if len(s) <= n:
        return s
    return s[: n - 1] + "…"
