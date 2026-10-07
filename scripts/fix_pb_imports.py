"""Rewrite protoc-generated imports to piper_agent.pb.* (run after gen_proto)."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PB = ROOT / "agents" / "src" / "piper_agent" / "pb"

REPLACEMENTS = [
    (re.compile(r"^from runtime\.v1 import ", re.MULTILINE), "from piper_agent.pb.runtime.v1 import "),
    (re.compile(r"^from common\.v1 import ", re.MULTILINE), "from piper_agent.pb.common.v1 import "),
    (re.compile(r"^from agent\.v1 import ", re.MULTILINE), "from piper_agent.pb.agent.v1 import "),
]


def fix_file(path: Path) -> bool:
    text = path.read_text(encoding="utf-8")
    new = text
    for pattern, repl in REPLACEMENTS:
        new = pattern.sub(repl, new, count=0)
    if new != text:
        path.write_text(new, encoding="utf-8")
        return True
    return False


def ensure_init_dirs() -> None:
    for rel in ("common", "common/v1", "runtime", "runtime/v1", "agent", "agent/v1"):
        d = PB / rel
        d.mkdir(parents=True, exist_ok=True)
        init = d / "__init__.py"
        if not init.exists():
            init.write_text("", encoding="utf-8")


def main() -> None:
    ensure_init_dirs()
    changed = 0
    for path in PB.rglob("*.py"):
        if fix_file(path):
            changed += 1
            print(f"fixed: {path.relative_to(ROOT)}")
    print(f"done, {changed} file(s) updated")


if __name__ == "__main__":
    main()
