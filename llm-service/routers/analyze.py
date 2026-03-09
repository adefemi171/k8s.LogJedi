"""POST /analyze endpoint; GET /analyses and /reports; POST /report-outcome for operator callback."""

import json
import logging
from datetime import datetime, timezone
from typing import Any

from fastapi import APIRouter, BackgroundTasks

from models import AnalyzeRequest, AnalyzeResponse, ReportOutcomeRequest
from services.analyzer import analyze_workload

router = APIRouter()
logger = logging.getLogger(__name__)

# In-memory cap for request/response log (simple ring: last N chars)
_LOG_CAP = 8000
_log_buffer: list[str] = []

# Analyses: summary/result per request (for GET /analyses)
_MAX_ANALYSES = 100
_analyses: list[dict[str, Any]] = []

# Reports: step-by-step log of what the LLM and operator did (for GET /reports)
_MAX_REPORTS = 100
_reports: list[dict[str, Any]] = []


def _redact_spec(spec: dict) -> dict:
    """Shallow redact for logging: truncate large spec."""
    if not spec:
        return {}
    s = json.dumps(spec)[:_LOG_CAP]
    return {"_truncated": len(json.dumps(spec)) > _LOG_CAP, "_preview": s[:500]}


def _get_containers_from_spec(spec: dict) -> list[Any]:
    """Get containers list from Pod spec or Deployment spec (spec.template.spec.containers)."""
    if not spec:
        return []
    # Pod: spec.containers
    c = spec.get("containers")
    if c:
        return c
    # Deployment/ReplicaSet: spec.spec.template.spec.containers or spec.template.spec.containers
    inner = spec.get("spec") or spec
    template = inner.get("template") or {}
    return template.get("spec", {}).get("containers") or []


def _describe_patch_step(req: AnalyzeRequest, resp: AnalyzeResponse) -> str:
    """Build a human-readable 'suggested patch' step from request spec and response action."""
    if not resp.action or not resp.action.patch:
        return "No patch suggested."
    spec = req.spec or {}
    patch = resp.action.patch or {}
    # Common case: image change in spec.containers
    patch_spec = patch.get("spec") or {}
    patch_containers = patch_spec.get("containers") or (patch_spec.get("template") or {}).get("spec", {}).get("containers") or []
    current_containers = _get_containers_from_spec(spec)
    current_by_name: dict[str, str] = {}
    for c in current_containers:
        if isinstance(c, dict) and c.get("name"):
            current_by_name[c["name"]] = str(c.get("image", ""))
    parts: list[str] = []
    for c in patch_containers:
        if not isinstance(c, dict):
            continue
        name = c.get("name", "")
        new_image = c.get("image")
        if new_image is None:
            continue
        old_image = current_by_name.get(name, "")
        if old_image and old_image != new_image:
            parts.append(f"container '{name}' image from {old_image} → {new_image}")
        else:
            parts.append(f"container '{name}' image set to {new_image}")
    if parts:
        return "Suggested patch: " + "; ".join(parts) + "."
    return "Suggested patch: " + json.dumps(patch)[:200] + ("..." if len(json.dumps(patch)) > 200 else ".")


def _build_report_steps(req: AnalyzeRequest, resp: AnalyzeResponse) -> list[dict[str, Any]]:
    """Build ordered steps for one analysis (LLM steps; operator step added later via POST /report-outcome)."""
    ts = datetime.now(timezone.utc).isoformat()
    steps: list[dict[str, Any]] = [
        {"step": 1, "timestamp": ts, "message": f"Received analysis request for {req.resource_kind} '{req.resource_name}' in namespace '{req.namespace}' (reason: {req.reason})."},
        {"step": 2, "timestamp": ts, "message": f"Root cause: {resp.root_cause or '(none)'}"},
        {"step": 3, "timestamp": ts, "message": f"Recommendation: {resp.recommendation or '(none)'}"},
    ]
    if resp.action and resp.action.type == "k8s_patch":
        steps.append({"step": 4, "timestamp": ts, "message": _describe_patch_step(req, resp)})
    else:
        steps.append({"step": 4, "timestamp": ts, "message": "No k8s patch suggested (manual remediation only)."})
    return steps


def _store_analysis(req: AnalyzeRequest, resp: AnalyzeResponse) -> None:
    """Append one analysis record and one step-by-step report."""
    try:
        ts = datetime.now(timezone.utc).isoformat()
        record = {
            "timestamp": ts,
            "resource_kind": req.resource_kind,
            "resource_name": req.resource_name,
            "namespace": req.namespace,
            "reason": req.reason,
            "summary": resp.summary or "",
            "root_cause": resp.root_cause or "",
            "recommendation": resp.recommendation or "",
            "action": resp.action.model_dump() if resp.action else None,
        }
        _analyses.append(record)
        if len(_analyses) > _MAX_ANALYSES:
            _analyses.pop(0)

        report = {
            "timestamp": ts,
            "resource_kind": req.resource_kind,
            "resource_name": req.resource_name,
            "namespace": req.namespace,
            "reason": req.reason,
            "steps": _build_report_steps(req, resp),
        }
        _reports.append(report)
        if len(_reports) > _MAX_REPORTS:
            _reports.pop(0)
    except Exception as e:
        logger.warning("store analysis/report: %s", e)


def _log_request_response(req: AnalyzeRequest, resp: AnalyzeResponse) -> None:
    """Background task: store analysis and log request/response with sensitive fields redacted."""
    _store_analysis(req, resp)
    try:
        entry = {
            "request": {
                "resource_kind": req.resource_kind,
                "resource_name": req.resource_name,
                "namespace": req.namespace,
                "reason": req.reason,
                "events_count": len(req.events),
                "nodes_count": len(req.nodes or []),
                "spec": _redact_spec(req.spec),
                "recent_logs_lines": len(req.recent_logs or []),
                "historical_logs_lines": len(req.historical_logs or []),
            },
            "response": {
                "summary": (resp.summary or "")[:_LOG_CAP],
                "root_cause": (resp.root_cause or "")[:500],
                "recommendation": (resp.recommendation or "")[:500],
                "has_action": resp.action is not None,
            },
        }
        msg = json.dumps(entry)[:_LOG_CAP]
        _log_buffer.append(msg)
        if len(_log_buffer) > 100:
            _log_buffer.pop(0)
        logger.info("analyze request/response: %s", msg)
    except Exception as e:
        logger.warning("log request/response: %s", e)


@router.post("/analyze", response_model=AnalyzeResponse)
async def analyze(req: AnalyzeRequest, background_tasks: BackgroundTasks) -> AnalyzeResponse:
    """Analyze a failed workload and return explanation and optional patch."""
    resp = await analyze_workload(req)
    background_tasks.add_task(_log_request_response, req, resp)
    return resp


@router.get("/analyses")
async def get_analyses(limit: int = 50) -> list[dict[str, Any]]:
    """Return the most recent analyses (summary, root cause, recommendation, action). Newest first."""
    n = min(max(1, limit), _MAX_ANALYSES)
    return list(reversed(_analyses[-n:]))


@router.get("/reports")
async def get_reports(limit: int = 50) -> list[dict[str, Any]]:
    """Return step-by-step reports: each report has 'steps' (received request, root cause, recommendation, suggested patch, and if applicable operator applied). Newest first."""
    n = min(max(1, limit), _MAX_REPORTS)
    return list(reversed(_reports[-n:]))


@router.post("/report-outcome")
async def report_outcome(body: ReportOutcomeRequest) -> dict[str, str]:
    """Called by the operator when it applies a patch. Appends an 'Operator applied patch' step to the latest matching report."""
    ts = datetime.now(timezone.utc).isoformat()
    resource_kind = body.resource_kind
    resource_name = body.resource_name
    namespace = body.namespace
    # Find latest report for this resource that doesn't already have an operator step
    for i in range(len(_reports) - 1, -1, -1):
        r = _reports[i]
        if (
            r.get("resource_kind") == resource_kind
            and r.get("resource_name") == resource_name
            and r.get("namespace") == namespace
        ):
            steps = r.get("steps") or []
            if any("Operator applied" in str(s.get("message", "")) for s in steps):
                continue
            steps.append(
                {
                    "step": len(steps) + 1,
                    "timestamp": ts,
                    "message": f"Operator applied patch to {resource_kind} '{resource_name}' in namespace '{namespace}'.",
                }
            )
            r["steps"] = steps
            logger.info("report-outcome: appended operator step for %s %s/%s", resource_kind, namespace, resource_name)
            return {"status": "ok", "message": "Appended operator step to report"}
    logger.warning("report-outcome: no matching report for %s %s/%s", resource_kind, namespace, resource_name)
    return {"status": "not_found", "message": "No matching report found to append operator step"}
