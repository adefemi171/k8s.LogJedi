"""Strands-based analyzer: builds prompt, calls LLM with structured output."""

import asyncio
import json
from typing import Optional

from config import (
    LLM_PROVIDER,
    LLM_API_KEY,
    OPENAI_MODEL_ID,
    AWS_REGION,
    BEDROCK_MODEL_ID,
    GEMINI_MODEL_ID,
)
from models import (
    AnalyzeRequest,
    AnalyzeResponse,
    K8sPatchAction,
    ActionTarget,
)

# Optional provider imports (strands and provider backends may not be installed)
try:
    from strands import Agent
except ImportError:
    Agent = None

try:
    from strands.models.openai import OpenAIModel
except ImportError:
    OpenAIModel = None

try:
    from strands.models.bedrock import BedrockModel
except ImportError:
    BedrockModel = None

try:
    from strands.models.gemini import GeminiModel
except ImportError:
    GeminiModel = None


def _build_prompt(req: AnalyzeRequest) -> str:
    """Build a single prompt for the LLM with clear recent vs historical log sections."""
    lines = [
        "You are a Kubernetes SRE. Analyze the following failed workload and provide:",
        "1. A short summary (human-readable explanation)",
        "2. Root cause",
        "3. A recommendation (natural language remediation)",
        "4. Optionally a safe Kubernetes patch (only image, env, replicas, or resources).",
        "",
        f"Resource: {req.resource_kind} {req.resource_name} in namespace {req.namespace}",
        f"Failure reason: {req.reason}",
        "",
        "--- Events ---",
    ]
    for e in req.events:
        lines.append(f"  [{e.type}] {e.reason}: {e.message}")
    lines.append("")
    lines.append("--- Spec (snippet) ---")
    lines.append(json.dumps(req.spec, indent=2)[:4000])
    lines.append("")
    lines.append("--- Recent logs (last few minutes) ---")
    for line in (req.recent_logs or [])[:200]:
        lines.append(line)
    if req.historical_logs:
        lines.append("")
        lines.append("--- Historical logs (last hour) ---")
        for line in req.historical_logs[:300]:
            lines.append(line)
    lines.append("")
    lines.append("Use the above to detect patterns (repeated errors, time windows). Produce summary, root_cause, recommendation, and if appropriate a k8s_patch action targeting the same resource.")
    lines.append("")
    lines.append("Important: If the failure is ImagePullBackOff, ErrImagePull, or any image pull error, the current image is wrong or unreachable. In your k8s_patch, set the container image to a known-pullable image such as nginx:latest or busybox:latest (do not invent registry paths like example.com/valid-image:latest). For other failure types, only suggest a patch if you can make a safe, concrete change (image, env, replicas, or resources).")
    return "\n".join(lines)


def _create_agent():
    """Create a Strands Agent with the model for the configured LLM_PROVIDER. Returns None if mock or unsupported."""
    if not LLM_PROVIDER or LLM_PROVIDER == "mock":
        return None
    if Agent is None:
        return None

    provider = LLM_PROVIDER.lower()

    # OpenAI
    if provider == "openai":
        if OpenAIModel is None:
            return None
        model = OpenAIModel(
            client_args={"api_key": LLM_API_KEY} if LLM_API_KEY else {},
            model_id=OPENAI_MODEL_ID,
            params={"max_tokens": 2000, "temperature": 0.3},
        )
        return Agent(model=model)

    # AWS Bedrock (uses boto3 / AWS credentials from env or IAM role)
    if provider == "bedrock":
        if BedrockModel is None:
            return None
        model = BedrockModel(
            model_id=BEDROCK_MODEL_ID,
            region_name=AWS_REGION or None,
        )
        return Agent(model=model)

    # Google Gemini (API key or GOOGLE_APPLICATION_CREDENTIALS for Vertex)
    if provider == "gemini":
        if GeminiModel is None:
            return None
        client_args = {}
        if LLM_API_KEY:
            client_args["api_key"] = LLM_API_KEY
        model = GeminiModel(
            client_args=client_args,
            model_id=GEMINI_MODEL_ID,
            params={"temperature": 0.3, "max_output_tokens": 2048},
        )
        return Agent(model=model)

    return None


async def analyze_workload(req: AnalyzeRequest) -> AnalyzeResponse:
    """Run LLM analysis: mock returns fixed response; real uses Strands agent with structured output."""
    if LLM_PROVIDER == "mock" or not LLM_PROVIDER:
        return _mock_response(req)

    agent = _create_agent()
    if agent is None:
        return _mock_response(req)

    prompt = _build_prompt(req)
    try:
        invoke_async = getattr(agent, "invoke_async", None)
        if invoke_async is not None:
            result = await invoke_async(
                prompt,
                structured_output_model=AnalyzeResponse,
            )
        else:
            def _run():
                return agent(prompt, structured_output_model=AnalyzeResponse)
            result = await asyncio.get_event_loop().run_in_executor(None, _run)
        out = getattr(result, "structured_output", None)
        if out is None:
            return _mock_response(req)
        return out
    except Exception:
        return _mock_response(req)


def _mock_response(req: AnalyzeRequest) -> AnalyzeResponse:
    """Fixed response for mock provider and fallback."""
    return AnalyzeResponse(
        summary="Mock analysis: the workload appears to be failing. Check events and logs above for details.",
        root_cause="Not determined (mock provider).",
        recommendation="Review recent logs and events; consider adjusting image, env, or resources.",
        action=K8sPatchAction(
            type="k8s_patch",
            target=ActionTarget(
                kind=req.resource_kind,
                namespace=req.namespace,
                name=req.resource_name,
            ),
            patch_type="application/strategic-merge-patch+json",
            patch={"spec": {"template": {"spec": {"containers": [{"name": "app", "image": "example:latest"}]}}}},
        ) if req.resource_kind == "Deployment" else None,
    )
