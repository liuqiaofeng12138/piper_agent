from __future__ import annotations

import json
import logging
from abc import ABC, abstractmethod
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

logger = logging.getLogger(__name__)

ProgressFn = Callable[[str, str], None]


@dataclass
class TaskContext:
    user_message: str
    documents: list[tuple[str, bytes]]
    skill_outputs: dict[str, str] = field(default_factory=dict)


@dataclass
class SkillDefinition:
    skill_id: str
    display_name: str
    description: str
    system_prompt: str


class Skill(ABC):
    definition: SkillDefinition

    @abstractmethod
    def openai_tools(self) -> list[dict[str, Any]]:
        raise NotImplementedError

    @abstractmethod
    def execute_tool(self, name: str, arguments: dict[str, Any], ctx: TaskContext) -> str:
        raise NotImplementedError

    def run(
        self,
        llm: Any,
        model: str,
        ctx: TaskContext,
        max_steps: int,
        on_progress: ProgressFn | None = None,
    ) -> str:
        user_block = (
            f"用户总需求:\n{ctx.user_message}\n\n"
            "请仅完成本技能职责范围内的子任务，输出结构化要点，便于主控汇总。"
        )
        return run_tool_loop(
            llm=llm,
            model=model,
            system=self.definition.system_prompt,
            user=user_block,
            tools=self.openai_tools(),
            execute=lambda name, args: self.execute_tool(name, args, ctx),
            max_steps=max_steps,
            skill_id=self.definition.skill_id,
            on_progress=on_progress,
        )


def run_tool_loop(
    *,
    llm: Any,
    model: str,
    system: str,
    user: str,
    tools: list[dict[str, Any]],
    execute: Callable[[str, dict[str, Any]], str],
    max_steps: int,
    skill_id: str,
    on_progress: ProgressFn | None,
) -> str:
    messages: list[dict[str, Any]] = [
        {"role": "system", "content": system},
        {"role": "user", "content": user},
    ]
    for step in range(max_steps):
        if on_progress:
            on_progress(skill_id, f"技能推理步骤 {step + 1}/{max_steps}")
        kwargs: dict[str, Any] = {"model": model, "messages": messages}
        if tools:
            kwargs["tools"] = tools
            kwargs["tool_choice"] = "auto"
        response = llm.chat.completions.create(**kwargs)
        choice = response.choices[0]
        msg = choice.message
        tool_calls = getattr(msg, "tool_calls", None) or []
        assistant_entry: dict[str, Any] = {
            "role": "assistant",
            "content": msg.content or "",
        }
        if tool_calls:
            assistant_entry["tool_calls"] = [
                {
                    "id": tc.id,
                    "type": "function",
                    "function": {
                        "name": tc.function.name,
                        "arguments": tc.function.arguments or "{}",
                    },
                }
                for tc in tool_calls
            ]
        messages.append(assistant_entry)

        if not tool_calls:
            return (msg.content or "").strip() or "（技能未返回文本）"

        for tc in tool_calls:
            fn = tc.function.name
            raw_args = tc.function.arguments or "{}"
            try:
                args = json.loads(raw_args) if isinstance(raw_args, str) else dict(raw_args)
            except json.JSONDecodeError:
                args = {}
            if on_progress:
                on_progress(skill_id, f"调用工具 {fn}")
            try:
                result = execute(fn, args)
            except Exception as e:  # noqa: BLE001
                logger.warning("tool %s failed: %s", fn, e)
                result = f"工具执行失败: {e}"
            messages.append(
                {
                    "role": "tool",
                    "tool_call_id": tc.id,
                    "content": result[:16000],
                }
            )

    return "（达到本技能最大工具步数，请缩小问题范围后重试）"
