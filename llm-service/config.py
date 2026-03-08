"""Configuration from environment variables. Loads .env if present."""

import os
from pathlib import Path

# Load .env: llm-service/.env first, then repo root .env (root is for shared keys e.g. OpenAI)
try:
    from dotenv import load_dotenv
except ImportError:
    load_dotenv = None
if load_dotenv:
    _dir = Path(__file__).resolve().parent
    _local = _dir / ".env"
    _root = _dir.parent / ".env"
    if _local.exists():
        load_dotenv(_local)
    if _root.exists():
        load_dotenv(_root)


def get_env(key: str, default: str = "") -> str:
    return os.environ.get(key, default).strip()


# LLM provider: mock | openai | bedrock | gemini
LLM_PROVIDER = get_env("LLM_PROVIDER", "mock")
LLM_API_KEY = get_env("LLM_API_KEY", "")

# OpenAI (when LLM_PROVIDER=openai)
OPENAI_MODEL_ID = get_env("OPENAI_MODEL_ID", "gpt-4o")

# AWS Bedrock (when LLM_PROVIDER=bedrock) – uses AWS credentials from env or IAM role
AWS_REGION = get_env("AWS_REGION", "us-west-2")
BEDROCK_MODEL_ID = get_env("BEDROCK_MODEL_ID", "us.anthropic.claude-sonnet-4-20250514-v1:0")

# Google Gemini (when LLM_PROVIDER=gemini) – API key or Vertex with GOOGLE_APPLICATION_CREDENTIALS
GEMINI_MODEL_ID = get_env("GEMINI_MODEL_ID", "gemini-2.0-flash")
# Optional: Vertex AI project/region (if using Vertex instead of AI Studio)
VERTEX_PROJECT_ID = get_env("VERTEX_PROJECT_ID", "")
CLOUD_ML_REGION = get_env("CLOUD_ML_REGION", "")

# Server
HOST = get_env("HOST", "0.0.0.0")
PORT = int(get_env("PORT", "8000"))
