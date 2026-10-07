from __future__ import annotations

from piper_agent.clients.runtime_client import RuntimeClient
from piper_agent.config.loader import AgentConfig, load_agent_config
from piper_agent.harness.loop import AgentLoop
from piper_agent.harness.registry import ToolRegistry
from piper_agent.harness.session import SessionState
from piper_agent.rag.template_index import TemplateIndex
from piper_agent.tools.handlers import ToolHandlers


def create_loop(config_path: str | None, runtime_addr: str | None) -> tuple[AgentLoop, RuntimeClient]:
    cfg = load_agent_config(config_path)
    if runtime_addr:
        cfg.runtime_address = runtime_addr
    api_key = cfg.llm.api_key
    if not api_key:
        hint = cfg.config_path or "deploy/config/local.yaml"
        raise RuntimeError(
            f"请在配置文件中设置 llm.api_key（OpenAI 或兼容服务的 API Key）: {hint}"
        )
    try:
        from openai import OpenAI
    except ImportError as e:
        raise RuntimeError("Install LLM extras: pip install -e '.[llm]'") from e

    kwargs: dict = {"api_key": api_key}
    if cfg.llm.base_url:
        kwargs["base_url"] = cfg.llm.base_url
    llm = OpenAI(**kwargs)

    client = RuntimeClient(address=cfg.runtime_address)
    session = SessionState()
    index = TemplateIndex(client, examples_dir=cfg.examples_dir)
    index.refresh()
    handlers = ToolHandlers(client, session, cfg, index)
    registry = ToolRegistry(handlers)
    loop = AgentLoop(registry, session, cfg, llm_client=llm)
    return loop, client
