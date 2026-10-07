from __future__ import annotations

from web_crawler_agent.harness.session import SessionState


class PolicyError(Exception):
    pass


def ensure_validated_before_run(session: SessionState, template_id: str, *, required: bool) -> None:
    if not required:
        return
    if template_id not in session.validated_template_ids:
        raise PolicyError(
            f"template {template_id} not validated in this session; call template_author_validate first"
        )


def ensure_run_quota(session: SessionState, max_runs: int) -> None:
    if max_runs <= 0:
        return
    if session.run_count >= max_runs:
        raise PolicyError(f"session run limit reached ({max_runs}); start a new chat session")
