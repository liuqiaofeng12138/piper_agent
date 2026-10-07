"""Rewrite protoc-generated imports to <pkg>.pb.* (run after gen_proto).

claw_agent（采集链路，含 runtime/common/agent 全部 stub）→ claw_agent.pb.*
agents（通用 Worker，仅 agent/v1 stub）→ piper_agent.pb.*
"""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

TARGETS = [
    (ROOT / "claw_agent" / "src" / "claw_agent" / "pb", "claw_agent"),
    (ROOT / "agents" / "src" / "piper_agent" / "pb", "piper_agent"),
]

IMPORT_RE = [
    (re.compile(r"^from runtime\.v1 import ", re.MULTILINE), "from {pkg}.pb.runtime.v1 import "),
    (re.compile(r"^from common\.v1 import ", re.MULTILINE), "from {pkg}.pb.common.v1 import "),
    (re.compile(r"^from agent\.v1 import ", re.MULTILINE), "from {pkg}.pb.agent.v1 import "),
]


def fix_file(path: Path, pkg: str) -> bool:
    text = path.read_text(encoding="utf-8")
    new = text
    for pattern, repl in IMPORT_RE:
        new = pattern.sub(repl.format(pkg=pkg), new, count=0)
    if new != text:
        path.write_text(new, encoding="utf-8")
        return True
    return False


def ensure_init_dirs(pb: Path) -> None:
    for rel in ("common", "common/v1", "runtime", "runtime/v1", "agent", "agent/v1"):
        d = pb / rel
        if not d.exists():
            continue
        init = d / "__init__.py"
        if not init.exists():
            init.write_text("", encoding="utf-8")


def main() -> None:
    changed = 0
    for pb, pkg in TARGETS:
        if not pb.is_dir():
            continue
        ensure_init_dirs(pb)
        for path in pb.rglob("*.py"):
            if fix_file(path, pkg):
                changed += 1
                print(f"fixed: {path.relative_to(ROOT)}")
    print(f"done, {changed} file(s) updated")


if __name__ == "__main__":
    main()
