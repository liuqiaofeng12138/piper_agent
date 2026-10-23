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
  [enterprise_agent]="dev"
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
  OFFICIAL_PIP=(-i "https://pypi.org/simple" --trusted-host pypi.org)
  pip_install_fallback() {
    local use_no_build=""
    local args=()
    for a in "$@"; do
      if [[ "$a" == "--no-build-isolation" ]]; then
        use_no_build=1
      else
        args+=("$a")
      fi
    done
    if [[ ${#PIP_MIRROR[@]} -gt 0 ]]; then
      if [[ -n "$use_no_build" ]]; then
        pip install "${PIP_MIRROR[@]}" "${args[@]}" --no-build-isolation && return 0
      else
        pip install "${PIP_MIRROR[@]}" "${args[@]}" && return 0
      fi
      echo "  镜像源 pip install 失败，尝试官方 PyPI ..."
    fi
    if [[ -n "$use_no_build" ]]; then
      pip install "${OFFICIAL_PIP[@]}" "${args[@]}" --no-build-isolation
    else
      pip install "${OFFICIAL_PIP[@]}" "${args[@]}"
    fi
  }
  echo "  升级 pip ..."
  pip_install_fallback -U pip
  echo "  安装构建依赖 setuptools>=68 ..."
  pip_install_fallback -U "setuptools>=68"
  echo "  安装/升级 wheel（可选）..."
  pip_install_fallback -U wheel || true
  extra="${EXTRAS[$name]:-}"
  echo "  安装项目依赖 ..."
  if [[ -n "$extra" ]]; then
    pip_install_fallback -e ".[$extra]" --no-build-isolation
  else
    pip_install_fallback -e . --no-build-isolation
  fi
  deactivate
done

echo "Done. Each agent uses its own <project>/.venv when started via piper-serve."
