"""Tests for analyze router (GET /analyses, GET /reports)."""

import pytest
from fastapi.testclient import TestClient

from main import app

client = TestClient(app)


def test_health():
    """GET /health returns 200."""
    r = client.get("/health")
    assert r.status_code == 200


def test_ready():
    """GET /ready returns 200."""
    r = client.get("/ready")
    assert r.status_code == 200


def test_analyses_returns_list():
    """GET /analyses returns a JSON list (may be empty)."""
    r = client.get("/analyses")
    assert r.status_code == 200
    data = r.json()
    assert isinstance(data, list)


def test_analyses_limit_param():
    """GET /analyses?limit=5 returns at most 5 items."""
    r = client.get("/analyses?limit=5")
    assert r.status_code == 200
    data = r.json()
    assert isinstance(data, list)
    assert len(data) <= 5


def test_reports_returns_list():
    """GET /reports returns a JSON list (may be empty)."""
    r = client.get("/reports")
    assert r.status_code == 200
    data = r.json()
    assert isinstance(data, list)


def test_reports_have_steps():
    """Each report item has timestamp, resource fields, and steps."""
    r = client.get("/reports?limit=1")
    assert r.status_code == 200
    data = r.json()
    if data:
        report = data[0]
        assert "timestamp" in report
        assert "resource_kind" in report
        assert "resource_name" in report
        assert "namespace" in report
        assert "steps" in report
        assert isinstance(report["steps"], list)
