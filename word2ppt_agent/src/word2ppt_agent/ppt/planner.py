from __future__ import annotations

import json
import logging
from typing import Any

from word2ppt_agent.config.loader import Word2PPTConfig
from word2ppt_agent.ppt.outline import DeckOutline, parse_outline_json

logger = logging.getLogger(__name__)

SYSTEM_PROMPT = """你是资深演示文稿策划、文案与版式顾问。根据用户上传的文档与制作目标，设计一份**信息密度适中、层次丰富、适合口头讲解**的 PPT 大纲。
只输出一段合法 JSON（不要 Markdown 代码块，不要额外说明）。

JSON 结构：
{
  "deck_title": "有吸引力的总标题（可含场景，如「技术面试自我介绍」）",
  "deck_subtitle": "一句话价值主张或听众收益",
  "theme": "corporate_blue|ocean|slate|warm",
  "slides": [
    { "layout": "title", "title": "...", "subtitle": "..." },
    { "layout": "agenda", "title": "本次汇报脉络", "bullets": ["章节1：…", "章节2：…"] },
    { "layout": "section", "title": "章节名", "subtitle": "本章一句话目标" },
    {
      "layout": "content",
      "title": "结论型小标题（动词或成果导向）",
      "subtitle": "本页导语：1 句承上启下",
      "bullets": ["要点：现象/做法 + 结果", "…"],
      "speaker_notes": "讲解时可强调的 1–2 句"
    },
    {
      "layout": "highlight",
      "title": "核心观点",
      "subtitle": "用 1–2 句话概括最关键结论（可稍长，适合大字展示）",
      "bullets": ["支撑论据1", "支撑论据2"]
    },
    {
      "layout": "metrics",
      "title": "关键数据或成果",
      "metrics": [
        { "label": "指标名", "value": "数值或描述" }
      ],
      "bullets": ["数据解读一句"]
    },
    {
      "layout": "two_column",
      "title": "对比/并列",
      "left_title": "左栏小标题",
      "right_title": "右栏小标题",
      "left_bullets": ["…"],
      "right_bullets": ["…"]
    },
    { "layout": "closing", "title": "总结与下一步", "bullets": ["回顾1", "行动建议", "Q&A"] }
  ]
}

内容与结构要求：
1. **忠于文档**：不编造文档中不存在的公司、项目、数字；无数据时 metrics 可省略或改用定性描述。
2. **页数**：不超过 max_slides；建议 14–22 页（含封面、目录、章节、结尾），章节 section 至少 2–3 个。
3. **必备页型**：title → agenda → 若干 section 开头 → 混合 content / highlight / metrics / two_column → closing。
4. **文案**：小标题用「结论先行」；要点用「关键词 + 短解释」；避免整段粘贴原文；面试/汇报场景用语专业、自信但不浮夸。
5. **节奏**：每章 2–4 页内容；重要结论用 highlight；有量化信息用 metrics（2–4 个指标）；对比结构用 two_column 并写清 left_title/right_title。
6. **theme**：技术/商务 → corporate_blue 或 slate；培训/创意 → ocean 或 warm。"""


def format_sources_for_llm(documents: list[tuple[str, str]], max_chars: int) -> str:
    parts: list[str] = []
    used = 0
    for name, text in documents:
        header = f"\n\n===== 文件：{name} =====\n"
        if used + len(header) >= max_chars:
            break
        parts.append(header)
        used += len(header)
        remain = max_chars - used
        chunk = text[:remain]
        parts.append(chunk)
        used += len(chunk)
        if len(text) > len(chunk):
            parts.append("\n[…文档后续内容已截断…]")
            break
    return "".join(parts).strip()


def plan_deck_outline(
    client: Any,
    model: str,
    user_message: str,
    documents: list[tuple[str, str]],
    cfg: Word2PPTConfig,
) -> DeckOutline:
    sources = format_sources_for_llm(documents, cfg.max_source_chars)
    user_prompt = (
        f"用户制作要求：{(user_message or '').strip() or '请根据上传文档生成专业汇总演示文稿'}\n\n"
        f"max_slides={cfg.max_slides_total}\n\n"
        f"文档内容：\n{sources}"
    )

    last_err: Exception | None = None
    for attempt in range(2):
        try:
            resp = client.chat.completions.create(
                model=model,
                messages=[
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": user_prompt},
                ],
                response_format={"type": "json_object"},
                temperature=0.55,
            )
            content = (resp.choices[0].message.content or "").strip()
            if not content:
                raise ValueError("大模型返回空内容")
            return parse_outline_json(content, cfg.max_slides_total)
        except (json.JSONDecodeError, ValueError, KeyError) as e:
            last_err = e
            logger.warning("outline parse failed attempt=%s: %s", attempt + 1, e)
            user_prompt += "\n\n上次 JSON 无效，请只输出符合 schema 的合法 JSON。"
        except Exception:
            raise
    raise ValueError(f"无法解析大模型大纲: {last_err}")
