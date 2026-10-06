from __future__ import annotations

import json
import time
from pathlib import Path
from typing import Any


def audit_log(path: Path | None, session_id: str, event: str, payload: dict[str, Any]) -> None:
    if path is None:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    row = {
        "ts": time.time(),
        "session_id": session_id,
        "event": event,
        **payload,
    }
    with path.open("a", encoding="utf-8") as f:
        f.write(json.dumps(row, ensure_ascii=False) + "\n")
