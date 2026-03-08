"""Health and readiness endpoints for the LLM service."""

from fastapi import APIRouter

router = APIRouter(tags=["health"])


@router.get("/health")
async def health() -> dict:
    """Liveness: is the process running."""
    return {"status": "ok"}


@router.get("/ready")
async def ready() -> dict:
    """Readiness: is the service ready to accept traffic (e.g. after startup)."""
    return {"status": "ready"}
