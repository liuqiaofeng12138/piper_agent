"""CLI entrypoint for Piper Agent (仅保留 doctor 连通性检查).

对话与任务执行请使用 Web：浏览器 → piper-gateway → Agent Worker。
"""

from __future__ import annotations

import argparse
from pathlib import Path

from claw_agent.clients.runtime_client import RuntimeClient
from claw_agent.config.loader import default_config_path, load_agent_config


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="claw-agent",
        description="Claw Agent 运维工具（对话请使用 Web 界面）",
    )
    sub = parser.add_subparsers(dest="command", required=True)
    default_cfg = str(default_config_path())

    doctor_p = sub.add_parser("doctor", help="Check Runtime gRPC connectivity")
    doctor_p.add_argument("--config", default=default_cfg)

    args = parser.parse_args()

    if args.command == "doctor":
        cfg = load_agent_config(args.config if Path(args.config).is_file() else None)
        client = RuntimeClient(address=cfg.runtime_address)
        try:
            client.list_templates(size=1, session_id="doctor")
            print(f"OK: Runtime reachable at {cfg.runtime_address}")
        except Exception as e:
            print(f"FAIL: cannot reach Runtime at {cfg.runtime_address}: {e}")
            print("Start: runtime/piper-runtime.exe -f ..\\deploy\\config\\local.yaml")
            raise SystemExit(1) from e
        finally:
            client.close()


if __name__ == "__main__":
    main()
