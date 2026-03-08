"""Pydantic request/response models for the analyze API."""

from typing import Any, Literal, Optional

from pydantic import BaseModel, Field, model_validator


class EventItem(BaseModel):
    """Single Kubernetes event."""

    type: str = Field(..., description="Normal or Warning")
    reason: str = ""
    message: str = ""
    firstTimestamp: Optional[str] = None
    lastTimestamp: Optional[str] = None


class AnalyzeRequest(BaseModel):
    """Request body for POST /analyze."""

    resource_kind: str = Field(..., description="Pod, Deployment, or Job")
    resource_name: str = ""
    namespace: str = ""
    reason: str = ""
    events: list[EventItem] = Field(default_factory=list)
    spec: dict[str, Any] = Field(default_factory=dict)
    recent_logs: list[str] = Field(default_factory=list, alias="recent_logs")
    historical_logs: list[str] = Field(default_factory=list, alias="historical_logs")

    model_config = {"populate_by_name": True}

    @model_validator(mode="before")
    @classmethod
    def coerce_list_fields(cls, data: Any) -> Any:
        """Accept null for list fields from Go (nil slices serialize as null) and coerce to []."""
        if not isinstance(data, dict):
            return data
        for key in ("recent_logs", "historical_logs", "events"):
            if key in data and data[key] is None:
                data[key] = []
        return data


class ActionTarget(BaseModel):
    """Target resource for a k8s patch action."""

    kind: str = Field(..., description="Deployment, Pod, or Job")
    namespace: str = ""
    name: str = ""


class K8sPatchAction(BaseModel):
    """Machine-readable patch action."""

    type: Literal["k8s_patch"] = "k8s_patch"
    target: ActionTarget
    patch_type: str = Field(default="application/strategic-merge-patch+json")
    patch: dict[str, Any] = Field(default_factory=dict)


class AnalyzeResponse(BaseModel):
    """Response body from POST /analyze."""

    summary: str = Field(..., description="Human-readable explanation")
    root_cause: str = ""
    recommendation: str = Field(..., description="Natural language remediation")
    action: Optional[K8sPatchAction] = None


class ReportOutcomeRequest(BaseModel):
    """Request body for POST /report-outcome (operator callback when patch is applied)."""

    resource_kind: str = Field(..., description="e.g. Deployment, Pod, Job")
    resource_name: str = Field(..., description="Name of the resource")
    namespace: str = Field(..., description="Namespace")
    outcome: str = Field(default="applied", description="e.g. applied")
