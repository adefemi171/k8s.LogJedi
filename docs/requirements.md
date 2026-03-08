# Requirements and acceptance criteria

This document captures high-level requirements and acceptance criteria for k8s LogJedi. It complements the [ROADMAP](ROADMAP.md) and [ADRs](adr/).

## User stories

### SRE / Operator

- **As an** SRE, **I want** failed workloads (e.g. ImagePullBackOff, CrashLoopBackOff) to be analyzed automatically **so that** I get a clear summary and suggested fix without manually reading events and logs.
- **As an** SRE, **I want** to choose between auto-apply of safe patches or manual review (e.g. Slack/Teams) **so that** I can balance speed and control.
- **As an** SRE, **I want** the operator to only suggest or apply changes in namespaces I allow **so that** I don’t risk changes in system or other teams’ namespaces.
- **As an** SRE, **I want** secrets and sensitive data redacted from what is sent to the LLM **so that** no credentials leak outside the cluster.
- **As an** SRE, **I want** a step-by-step report and an analyses view of what the LLM did **so that** I can audit and debug.

### Platform / Security

- **As a** platform owner, **I want** the operator to use a least-privilege RBAC **so that** blast radius is limited.
- **As a** platform owner, **I want** optional authentication (e.g. Bearer token) between the operator and the LLM service **so that** only my operator can call the service.

## Acceptance criteria (MVP)

| ID | Requirement | Acceptance criteria |
|----|-------------|---------------------|
| AC1 | Failure detection | Operator detects CrashLoopBackOff, ImagePullBackOff, ErrImagePull, OOMKilled, failed Jobs, and unavailable Deployments for watched resources. |
| AC2 | Context collection | Operator collects Kubernetes Events and recent pod logs (and optional historical logs via configured backend) for the failed resource. |
| AC3 | Redaction | Specs sent to the LLM have env vars with secret-like names and valueFrom (secretKeyRef/configMapKeyRef) redacted. |
| AC4 | LLM analysis | LLM service returns JSON with summary, root_cause, recommendation, and optional k8s_patch action. |
| AC5 | Apply or notify | If APPLY_MODE=auto and patch is in allowed scope, operator applies patch; otherwise notifies (Slack/Teams) and/or logs suggested kubectl patch. |
| AC6 | Scope control | Operator respects WATCH_NAMESPACES, EXCLUDE_NAMESPACES, and AUTO_APPLY_NAMESPACES. |
| AC7 | Cooldown | Same resource is not re-analyzed within ANALYZE_COOLDOWN_MINUTES. |
| AC8 | Reports | GET /analyses and GET /reports return recent analyses and step-by-step reports; operator can report outcome so “Operator applied” appears in reports. |

## Out of scope (current)

- Formal SLOs (e.g. “analysis within 2 minutes”).
- Multi-tenant or per-team policy (beyond namespace filters).
- Persistent audit store (beyond operator logs and in-memory reports).
- Official support matrix (k8s versions, providers) – documented best-effort in README.

Requirements and acceptance criteria will be updated as the [ROADMAP](ROADMAP.md) evolves.
