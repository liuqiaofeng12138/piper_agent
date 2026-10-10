#!/usr/bin/env bash
# 为每个 Python 子 Agent 创建独立 .venv 并安装依赖
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="${PIPER_SETUP_PYTHON:-python3}"

declare -A EXTRAS=(
  [web_crawler_agent]="dev,llm"
  [general_agent]="llm"
  [rag_agent]="dev"
  [paper_agent]="dev"
)

for proj in "$ROOT"/*/pyproject.toml; do
  dir="$(dirname "$proj")"
  name="$(basename "$dir")"
  echo "==> $name"
  if [[ ! -d "$dir/.venv" ]]; then
    "$PY" -m venv "$dir/.venv"
  fi
  # shellcheck disable=SC1091
  source "$dir/.venv/bin/activate"
  extra="${EXTRAS[$name]:-}"
  if [[ -n "$extra" ]]; then
    pip install -e ".[$extra]"
  else
    pip install -e .
  fi
  deactivate
done

echo "Done. Each agent uses its own <project>/.venv when started via piper-serve."
