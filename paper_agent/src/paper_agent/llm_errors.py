"""将 OpenAI 兼容 SDK 异常转为面向用户的中文说明。"""

from __future__ import annotations


def format_llm_error(exc: BaseException) -> str:
    try:
        from openai import APIStatusError, AuthenticationError, RateLimitError
    except ImportError:
        return str(exc)

    if isinstance(exc, AuthenticationError):
        return (
            "大模型 API Key 无效或未授权。请检查 deploy/config/local.yaml 中的 "
            "llm.api_key 与 llm.base_url 是否与服务商一致。"
        )
    if isinstance(exc, RateLimitError):
        return "大模型请求过于频繁或配额用尽，请稍后重试。"
    if isinstance(exc, APIStatusError):
        code = exc.status_code
        if code == 402:
            return (
                "大模型 API 账户余额不足（HTTP 402 Insufficient Balance）。"
                "请在 DeepSeek/OpenAI 等服务商控制台充值，或更换有余额的 llm.api_key。"
            )
        if code == 401:
            return "大模型 API 未授权（HTTP 401），请检查 llm.api_key。"
        if code == 429:
            return "大模型请求限流或配额用尽（HTTP 429），请稍后重试。"
        if code == 404:
            return (
                f"大模型接口或模型不存在（HTTP 404）。请检查 llm.model（当前配置可能无效）"
                f" 与 llm.base_url。原始信息: {exc.message}"
            )
        body = getattr(exc, "message", None) or str(exc)
        return f"大模型 API 调用失败（HTTP {code}）：{body}"
    return str(exc)


def is_expected_llm_client_error(exc: BaseException) -> bool:
    try:
        from openai import APIStatusError, AuthenticationError, RateLimitError
    except ImportError:
        return False
    return isinstance(exc, (APIStatusError, AuthenticationError, RateLimitError))
