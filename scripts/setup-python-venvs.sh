#!/usr/bin/env bash
# 为每个 Python 子 Agent 创建独立 .venv 并安装依赖
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PY="${PIPER_SETUP_PYTHON:-python3}"

# 国内 PyPI 镜像（默认清华）。覆盖：PIPER_PIP_INDEX / PIP_INDEX_URL；官方源：PIPER_PIP_NO_MIRROR=1
DEFAULT_PIP_INDEX="https://pypi.tuna.tsinghua.edu.cn/simple"
DEFAULT_PIP_TRUSTED="pypi.tuna.tsinghua.edu.cn"

pip_mirror_args() {
  if [[ "${PIPER_PIP_NO_MIRROR:-}" == "1" ]]; then
    return
  fi
  local index="${PIPER_PIP_INDEX:-${PIP_INDEX_URL:-$DEFAULT_PIP_INDEX}}"
  local trusted="${PIPER_PIP_TRUSTED_HOST:-$DEFAULT_PIP_TRUSTED}"
  if [[ -z "$trusted" ]]; then
    trusted="$DEFAULT_PIP_TRUSTED"
  fi
  echo -i "$index" --trusted-host "$trusted"
}

declare -A EXTRAS=(
  [web_crawler_agent]="dev,llm"
  [general_agent]="llm"
  [rag_agent]="dev"
  [paper_agent]="dev"
  [word2ppt_agent]="dev"
)

if [[ "${PIPER_PIP_NO_MIRROR:-}" == "1" ]]; then
  echo "pip 镜像: pypi.org（官方，PIPER_PIP_NO_MIRROR=1）"
else
  echo "pip 镜像: ${PIPER_PIP_INDEX:-${PIP_INDEX_URL:-$DEFAULT_PIP_INDEX}}"
fi

# shellcheck disable=SC2207
PIP_MIRROR=($(pip_mirror_args))

for proj in "$ROOT"/*/pyproject.toml; do
  dir="$(dirname "$proj")"
  name="$(basename "$dir")"
  echo "==> $name"
  if [[ ! -d "$dir/.venv" ]]; then
    echo "  正在创建 .venv ..."
    "$PY" -m venv "$dir/.venv"
  fi
  # shellcheck disable=SC1091
  source "$dir/.venv/bin/activate"
  echo "  升级 pip / wheel ..."
  pip install "${PIP_MIRROR[@]}" -U pip wheel
  extra="${EXTRAS[$name]:-}"
  echo "  安装项目依赖 ..."
  if [[ -n "$extra" ]]; then
    pip install "${PIP_MIRROR[@]}" -e ".[$extra]"
  else
    pip install "${PIP_MIRROR[@]}" -e .
  fi
  deactivate
done

echo "Done. Each agent uses its own <project>/.venv when started via piper-serve."
