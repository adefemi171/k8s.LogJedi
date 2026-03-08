# k8s LogJedi Roadmap

**May the logs be with you.** Your AI SRE Jedi that reads failures, writes fixes.

This document outlines the current state of k8s LogJedi and planned improvements. It is intended for contributors and operators who want to extend or adopt the system.

---

## Current state (v0.1 – MVP)

The following is implemented and usable today:

### Operator and LLM service

- **Go operator:** controller-runtime; watches Pods, Deployments, Jobs; detects failure signals (CrashLoopBackOff, ImagePullBackOff, ErrImagePull, OOMKilled, failed Jobs, unavailable Deployments); collects Events and recent pod logs; redacts secrets from spec; POSTs to LLM service; applies patch (auto) or notifies (manual). **Resource limits** set in deploy (requests/limits for memory and CPU).
- **LLM service:** FastAPI `POST /analyze` with Pydantic validation; Strands agent with structured output. **Multiple LLM providers:** `mock` (default), `openai`, `bedrock`, `gemini`, configured via `LLM_PROVIDER` and provider-specific env (see README). **.env loading:** `python-dotenv` loads `llm-service/.env` for API keys and provider config. **Health:** `GET /health` (liveness), `GET /ready` (readiness). **OpenAPI:** `GET /openapi.json`. **Resource limits** in deploy manifest.
- **Log backend:** Pluggable interface with in-cluster Kubernetes backend and generic HTTP backend (query params: namespace, pod, since_minutes).
- **Notifications:** Slack (incoming webhook) and Microsoft Teams (message card); payload includes summary, root cause, recommendation, and proposed patch.

### Apply logic and safety

- **Apply logic:** `APPLY_MODE=auto` applies filtered strategic-merge patches (Deployment/Job/Pod); `manual` sends to Slack/Teams and logs `kubectl patch`. Scope limited to same namespace and allowed fields (image, env, replicas, resources). **Dry-run before apply:** optional server-side dry-run (`DRY_RUN_BEFORE_APPLY`) before applying; only apply if dry-run succeeds.
- **Cooldown:** Per-resource cooldown (`ANALYZE_COOLDOWN_MINUTES`, default 15) so the same resource is not re-analyzed within that window.
- **Prompt / log size limits:** `MAX_RECENT_LOG_LINES` and `MAX_HISTORICAL_LOG_LINES` (0 = no cap) cap payload sent to the LLM.
- **Namespace and apply scope:** `WATCH_NAMESPACES`, `EXCLUDE_NAMESPACES`, and `AUTO_APPLY_NAMESPACES` (comma-separated) restrict which namespaces are watched and where auto-apply is allowed.
- **Operator–LLM auth:** Optional `LLM_SERVICE_AUTH_HEADER` (e.g. Bearer token) for operator → LLM service calls.
- **Audit:** Operator logs "audit: applied patch to deployment/job/pod" when a patch is applied in auto mode.
- **Stronger redaction:** Redacts env values whose names suggest secrets; redacts `valueFrom` (secretKeyRef/configMapKeyRef) so the LLM never sees secret references.

### Docs and tooling

- **Deploy:** RBAC, ConfigMap, Deployment manifests; sample failing deployment; README quickstart for kind/minikube.
- **Runbook:** [RUNBOOK.md](RUNBOOK.md) for operator not reconciling, LLM returns no action, Slack/Teams not receiving, LLM unreachable, patch apply failures.
- **ADRs:** [docs/adr/](docs/adr/) for operator/LLM split and strategic merge patch.
- **Makefile:** `make build`, `make docker-build`, `make deploy`, `make test` (operator: go vet + go test; llm-service: pytest if available), `make dev` (build + docker-build with instructions for cluster + deploy).

---

## Near term (stability and production readiness)

| Area | Description |
|------|-------------|
| **Strands reliability** | Add retries and fallback when Strands structured output fails; optional timeout for agent invocation. |
| **Operator logging** | Use structured logging (e.g. logr) consistently; add request IDs or resource UIDs for tracing analyze requests. |
| **Tests** | Unit tests for failure detection, redaction, and patch filtering; integration test that runs operator + stub LLM in-process or in-cluster. |

---

## Mid term (features and integrations)

### Security and audit

| Area | Description |
|------|-------------|
| **Audit trail persistence** | Beyond operator logs: record when the operator applies a patch (resource, namespace, patch hash, timestamp) in a CRD, log stream, or external audit service. |
| **Further redaction** | Redact image pull secret names and volume paths that may expose secrets (in addition to current env and valueFrom redaction). |

### Scope and policy

| Area | Description |
|------|-------------|
| **Label filters** | Restrict which resources the operator watches by label (in addition to existing namespace filters). |
| **Cost guardrails** | Optional cap on analyze requests per hour (or per namespace) to avoid LLM cost blow-up. |

### Observability

| Area | Description |
|------|-------------|
| **Metrics** | Expose Prometheus metrics: analyze requests (success/failure), patch apply count, notification errors. |
| **Distributed tracing** | OpenTelemetry (or similar) so one trace links operator reconcile → HTTP to LLM → Strands/LLM call. |

### LLM and prompts

| Area | Description |
|------|-------------|
| **Configurable prompts** | Move Kubernetes SRE system prompt out of code into config (ConfigMap or file) for tuning without redeploy. |
| **Interactive approval (Slack/Teams)** | Buttons or actions in notifications to “Approve” or “Reject” a patch; operator applies only after approval (webhook or callback). |
| **Loki adapter** | Small service or option that translates `namespace`/`pod`/`since` into Loki LogQL and returns lines. |
| **More workload types** | Watch StatefulSets, DaemonSets, CronJobs; extend failure detection and patch allowlist. |
| **Config from ConfigMap** | Document and optionally extend to reload ConfigMap without restart. |
| **Log backend auth** | Support optional headers or basic auth for HTTP log backend. |

### Notifications

| Area | Description |
|------|-------------|
| **PagerDuty / Opsgenie / email** | Implement `Notifier` for PagerDuty, Opsgenie, or email so on-call gets analyses where they already are. |
| **Severity or routing** | Optional routing by severity (e.g. only PagerDuty for “critical” or certain namespaces). |

### Packaging and distribution

| Area | Description |
|------|-------------|
| **Helm chart** | Add a Helm chart (e.g. `charts/logjedi/`) that deploys both the operator and the LLM service with a single `helm install`. Expose key options as values: namespace, image tags, `LLM_SERVICE_URL`, Slack/Teams webhooks, `APPLY_MODE`, namespace filters, cooldown, log backend, etc. Enables one-command install and easier upgrades. |

---

## Long term / exploratory

| Area | Description |
|------|-------------|
| **LogJediAnalysis CRD** | Store analysis results in a custom resource for audit and UI; optional controller to sync from CR to Slack/Teams or apply. |
| **Multi-cluster** | Operator that can target multiple clusters and a single LLM service. |
| **Web UI** | Dashboard to list analyses, view proposed patches, approve/reject. |
| **Additional actions** | Beyond `k8s_patch`: e.g. full manifest, scaling, or script (with strong safety and review). |

---

## Developer and CI experience

| Area | Description |
|------|-------------|
| **Single-command local stack** | Extend `make dev` (or add Tilt/Skaffold) to optionally bring up kind, deploy, and create sample failing workload in one command. |
| **E2E test** | E2E test in kind: deploy operator + LLM + failing workload, assert analysis is produced (and optionally notification or patch). |
| **CI checks** | In CI: `make test` (operator: go vet + go test; llm-service: pytest); optional `make docker-build`. Add ruff/linting for LLM service. |

---

## Versioning and releases

- **Releases** will be tagged (e.g. `v0.2.0`) when a set of roadmap items is done; changelog can live in `CHANGELOG.md`.
- **Compatibility:** The operator and LLM service communicate via the documented JSON request/response schema; adding optional fields is backward compatible; changing required fields will be done with a version bump.

---

## Contributing

If you want to work on a roadmap item:

1. Open an issue describing the change and how it fits the roadmap.
2. Prefer small, reviewable PRs (e.g. one item or sub-item per PR).
3. Keep the README and this ROADMAP updated when adding config options or behavior.
