"""Tests for Pydantic models (request/response and null coercion)."""

import pytest
from pydantic import ValidationError

from models import AnalyzeRequest, AnalyzeResponse, EventItem, K8sPatchAction, ActionTarget


def test_analyze_request_coerce_null_lists():
    """Null for recent_logs, historical_logs, events should be coerced to []."""
    req = AnalyzeRequest(
        resource_kind="Pod",
        resource_name="test-pod",
        namespace="default",
        reason="ErrImagePull",
        recent_logs=None,
        historical_logs=None,
        events=None,
    )
    assert req.recent_logs == []
    assert req.historical_logs == []
    assert req.events == []


def test_analyze_request_from_dict_null_lists():
    """When parsing from dict (e.g. JSON from Go), null lists become []."""
    data = {
        "resource_kind": "Deployment",
        "resource_name": "app",
        "namespace": "default",
        "reason": "CrashLoopBackOff",
        "recent_logs": None,
        "historical_logs": None,
        "events": None,
        "spec": {},
    }
    req = AnalyzeRequest.model_validate(data)
    assert req.recent_logs == []
    assert req.historical_logs == []
    assert req.events == []


def test_analyze_request_requires_resource_kind():
    """resource_kind is required."""
    with pytest.raises(ValidationError):
        AnalyzeRequest(
            resource_name="x",
            namespace="default",
            reason="",
        )


def test_analyze_response_action_optional():
    """Response can have action null."""
    resp = AnalyzeResponse(
        summary="Test",
        root_cause="",
        recommendation="Fix it",
        action=None,
    )
    assert resp.action is None


def test_analyze_response_with_action():
    """Response can include k8s_patch action."""
    resp = AnalyzeResponse(
        summary="Test",
        root_cause="Image pull failed",
        recommendation="Use a valid image",
        action=K8sPatchAction(
            type="k8s_patch",
            target=ActionTarget(kind="Pod", namespace="default", name="x"),
            patch_type="application/strategic-merge-patch+json",
            patch={"spec": {"containers": [{"name": "app", "image": "nginx:latest"}]}},
        ),
    )
    assert resp.action is not None
    assert resp.action.target.name == "x"
    assert resp.action.patch["spec"]["containers"][0]["image"] == "nginx:latest"


def test_event_item_defaults():
    """EventItem has optional fields with defaults."""
    e = EventItem(type="Warning", reason="Failed", message="Something failed")
    assert e.firstTimestamp is None
    assert e.lastTimestamp is None
