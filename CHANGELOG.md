# Changelog

All notable changes to k8s LogJedi are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- See [ROADMAP.md](ROADMAP.md) for planned work.

## [0.1.0] - Initial release

### Added

- Go operator: controller-runtime; watches Pods, Deployments, Jobs; detects failure signals (CrashLoopBackOff, ImagePullBackOff, ErrImagePull, OOMKilled, failed Jobs, unavailable Deployments).
- LLM service: FastAPI `POST /analyze` with Pydantic validation; Strands agent with structured output; multi-provider support (mock, OpenAI, Bedrock, Gemini).
- Log backend: in-cluster Kubernetes and generic HTTP backend.
- Notifications: Slack and Microsoft Teams webhooks.
- Apply logic: `APPLY_MODE=auto` or `manual`; strategic-merge patch with dry-run option; per-resource cooldown; namespace and apply-scope filters.
- Secret redaction in specs sent to the LLM; optional operator–LLM auth header.
- GET /analyses and GET /reports; POST /report-outcome for operator callback (step-by-step reports).
- Helm chart under `charts/logjedi/` with configurable values.
- README, RUNBOOK, ROADMAP, ADRs, and deploy manifests.

[Unreleased]: https://github.com/adefemi171/k8s.LogJedi/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/adefemi171/k8s.LogJedi/releases/tag/v0.1.0
