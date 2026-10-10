"""OpenAI-compatible tool schemas for Phase C."""

TOOL_SCHEMAS: list[dict] = [
    {
        "type": "function",
        "function": {
            "name": "site_probe",
            "description": (
                "Probe a target URL over HTTP(S) before writing templates: final URL, "
                "content type, JSON keys/JSONPath hints or HTML title/SPA signals. "
                "Call this first for new sites, then combine with search_similar_templates."
            ),
            "parameters": {
                "type": "object",
                "properties": {
                    "url": {
                        "type": "string",
                        "description": "Page or API URL to fetch (http/https)",
                    },
                    "max_body_bytes": {
                        "type": "integer",
                        "description": "Max response bytes to read (default from harness config)",
                    },
                    "timeout_seconds": {
                        "type": "number",
                        "description": "HTTP timeout in seconds",
                    },
                },
                "required": ["url"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "search_similar_templates",
            "description": "RAG search over meta templates and shared/examples/templates/*.json",
            "parameters": {
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Keywords or user intent"},
                    "top_k": {"type": "integer", "default": 5},
                    "prefer_shared": {
                        "type": "boolean",
                        "default": False,
                        "description": "When true or query mentions shared/示例, rank example files above meta",
                    },
                },
                "required": ["query"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "load_shared_example_template",
            "description": "Load exact template JSON from shared/examples/templates (by id, stem, or filename). Always use returned template_json verbatim for validate/save.",
            "parameters": {
                "type": "object",
                "properties": {
                    "example_id": {
                        "type": "string",
                        "description": "Template id (e.g. http_jsonpath) or file stem",
                    },
                    "filename": {
                        "type": "string",
                        "description": "Optional file name e.g. http_jsonpath.json",
                    },
                },
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "template_author_validate",
            "description": "Validate template JSON via Runtime (structure + HTTP build)",
            "parameters": {
                "type": "object",
                "properties": {
                    "template_json": {"type": "string", "description": "Full template object as JSON string"},
                    "template_id": {"type": "string", "description": "Optional id for meta-only validate"},
                    "vars": {
                        "type": "object",
                        "additionalProperties": {"type": "string"},
                        "description": "Vars for {{placeholder}} replacement during validate",
                    },
                },
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "template_author_save",
            "description": "Upsert template to meta after successful validation",
            "parameters": {
                "type": "object",
                "properties": {
                    "template_json": {"type": "string"},
                    "template_id": {"type": "string"},
                },
                "required": ["template_json"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "param_filler_suggest",
            "description": "Suggest template vars from user natural language (heuristic)",
            "parameters": {
                "type": "object",
                "properties": {
                    "user_text": {"type": "string"},
                    "template_json": {"type": "string", "description": "Optional template to scan for {{var}} names"},
                },
                "required": ["user_text"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "list_proxies",
            "description": "List proxies from Piper meta for select_proxy",
            "parameters": {
                "type": "object",
                "properties": {
                    "status": {"type": "string", "description": "Optional filter e.g. Idle"},
                    "size": {"type": "integer", "default": 20},
                },
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "select_proxy",
            "description": "Bind a proxy for subsequent runner_execute in this session",
            "parameters": {
                "type": "object",
                "properties": {
                    "proxy_id": {"type": "string"},
                },
                "required": ["proxy_id"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "runner_execute",
            "description": "Run a validated template and return run_id (token id)",
            "parameters": {
                "type": "object",
                "properties": {
                    "template_id": {"type": "string"},
                    "vars": {"type": "object", "additionalProperties": {"type": "string"}},
                    "engine": {
                        "type": "string",
                        "enum": ["http", "chrome", "auto"],
                        "default": "auto",
                        "description": "auto picks from template builder.type",
                    },
                    "proxy_id": {"type": "string", "description": "Override session selected proxy"},
                },
                "required": ["template_id"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "runner_wait_for_complete",
            "description": "Block until last run finishes (SubscribeRun) and return final phase",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_id": {"type": "string", "description": "Defaults to last run"},
                },
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "runner_get_data",
            "description": "Fetch aggregated token data (docs/sources) after run",
            "parameters": {
                "type": "object",
                "properties": {
                    "token_id": {"type": "string", "description": "Defaults to last run"},
                },
            },
        },
    },
]
