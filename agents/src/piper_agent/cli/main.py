"""CLI entrypoint for Piper Agent."""

from __future__ import annotations

import argparse
from pathlib import Path

from piper_agent.clients.runtime_client import RuntimeClient
from piper_agent.config.loader import default_config_path, load_agent_config
from piper_agent.harness.repl import HarnessREPL


def _runtime_from_config(config_path: str | None) -> str:
    cfg = load_agent_config(config_path)
    return cfg.runtime_address


def main() -> None:
    parser = argparse.ArgumentParser(prog="piper-agent")
    sub = parser.add_subparsers(dest="command", required=True)
    default_cfg = str(default_config_path())

    repl_p = sub.add_parser("repl", help="Manual harness REPL (gRPC tools)")
    repl_p.add_argument("--config", default=default_cfg, help="deploy/config/local.yaml")
    repl_p.add_argument(
        "--runtime",
        default=None,
        help="override runtime.address from config",
    )

    chat_p = sub.add_parser("chat", help="Phase C LLM agent (OpenAI-compatible)")
    chat_p.add_argument("--config", default=default_cfg, help="deploy/config/local.yaml")
    chat_p.add_argument(
        "--runtime",
        default=None,
        help="override runtime.address from config",
    )

    doctor_p = sub.add_parser("doctor", help="Check Runtime gRPC connectivity")
    doctor_p.add_argument("--config", default=default_cfg)

    ask_p = sub.add_parser("ask", help="Single-turn chat (Phase D)")
    ask_p.add_argument("prompt", help="Natural language instruction")
    ask_p.add_argument("--config", default=default_cfg)
    ask_p.add_argument("--runtime", default=None)

    args = parser.parse_args()

    if args.command == "doctor":
        cfg = load_agent_config(args.config if Path(args.config).is_file() else None)
        client = RuntimeClient(address=cfg.runtime_address)
        try:
            client.list_templates(size=1, session_id="doctor")
            print(f"OK: Runtime reachable at {cfg.runtime_address}")
        except Exception as e:
            print(f"FAIL: cannot reach Runtime at {cfg.runtime_address}: {e}")
            print("Start: piper_agent/runtime/piper-runtime.exe -f ..\\deploy\\config\\local.yaml")
            raise SystemExit(1) from e
        finally:
            client.close()
        return

    config_path = args.config if Path(getattr(args, "config", "")).is_file() else None
    runtime_override = getattr(args, "runtime", None)
    runtime = runtime_override or _runtime_from_config(config_path or getattr(args, "config", None))

    if args.command == "repl":
        client = RuntimeClient(address=runtime)
        try:
            HarnessREPL(client).run()
        finally:
            client.close()
    elif args.command == "chat":
        from piper_agent.agents.orchestrator import chat_repl

        chat_repl(config_path or args.config, args.runtime)
    elif args.command == "ask":
        from piper_agent.agents.orchestrator import create_loop

        loop, client = create_loop(
            args.config if Path(args.config).is_file() else None, args.runtime
        )
        try:
            print(loop.run_turn(args.prompt))
        finally:
            client.close()


if __name__ == "__main__":
    main()
