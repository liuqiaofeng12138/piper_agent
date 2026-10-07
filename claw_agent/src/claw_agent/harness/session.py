from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from typing import Any


@dataclass
class SessionState:
    session_id: str = field(default_factory=lambda: f"sess-{uuid.uuid4().hex[:10]}")
    messages: list[dict[str, Any]] = field(default_factory=list)
    validated_template_ids: set[str] = field(default_factory=set)
    last_template_id: str | None = None
    last_run_id: str | None = None
    last_token_id: str | None = None
    pending_template: dict[str, Any] | None = None
    selected_proxy_id: str | None = None
    run_count: int = 0
