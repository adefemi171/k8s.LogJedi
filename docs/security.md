# Security and data flow

This document summarizes how k8s LogJedi handles data and security. See also [SECURITY.md](../SECURITY.md) for reporting vulnerabilities.

## Data flow

1. **Operator** (in-cluster) watches Pods, Deployments, Jobs. On failure it:
   - Reads Events and pod logs from the Kubernetes API.
   - Optionally fetches historical logs from a configured HTTP backend (e.g. Loki adapter).
   - Builds a JSON payload containing resource metadata, events, spec (redacted), and log lines.
   - Sends the payload to the **LLM service** over HTTP (optionally with an auth header).

2. **LLM service** receives the payload, runs an LLM (mock or provider such as OpenAI/Bedrock/Gemini), and returns:
   - Human-readable summary, root cause, recommendation.
   - Optional machine-readable patch (e.g. container image change).

3. **Operator** either applies the patch (if APPLY_MODE=auto and scope allows) or sends a notification (Slack/Teams) and logs a suggested `kubectl patch` command.

**Trust boundary:** The LLM service is not trusted with raw cluster secrets. Redaction happens in the operator before any data is sent. API keys for the LLM provider are stored in Kubernetes Secrets or env and never committed.

## Redaction

The operator redacts the following before sending to the LLM:

- **Env vars:** Any env var whose name (case-insensitive) contains SECRET, TOKEN, PASSWORD, KEY, or CREDENTIAL; the value is replaced with `REDACTED`.
- **valueFrom:** All `valueFrom` (secretKeyRef / configMapKeyRef) are removed or replaced so the LLM never sees secret or config map names.
- **imagePullSecrets:** Names are replaced with a placeholder.
- **Secret volumes:** Secret volume names are redacted.

See `operator/internal/redact/redact.go` for the implementation. Additional redaction (e.g. volume paths) is on the [ROADMAP](../ROADMAP.md).

## Authentication and transport

- **Operator → LLM:** Optional `LLM_SERVICE_AUTH_HEADER` (e.g. Bearer token) is sent on every request. In production, use TLS (HTTPS) and a strong token or mTLS.
- **LLM → provider:** Provider API keys are loaded from env or Kubernetes Secrets. They are not logged or sent to the operator.
- **Kubernetes API:** The operator uses in-cluster auth (ServiceAccount token) and RBAC. The ClusterRole is limited to get/list/watch/patch on pods, deployments, jobs, and events.

## RBAC and scope

- The operator runs as a dedicated ServiceAccount with a single ClusterRole. It can only read/write the resource types it needs (Pods, Deployments, Jobs, Events).
- **AUTO_APPLY_NAMESPACES** restricts where patches are applied (e.g. only `default`), reducing risk to system namespaces.
- **WATCH_NAMESPACES** and **EXCLUDE_NAMESPACES** limit which namespaces are watched for failures.

## Dependency and supply chain

- Go: `operator/go.mod` and `go.sum`; use `go mod verify` and keep modules updated (e.g. via Dependabot).
- Python: `llm-service/requirements.txt` and `pyproject.toml`; pin versions where appropriate and review updates.
- Container images: Built from the Dockerfiles in the repo. Consider image scanning (e.g. Trivy) in CI for production.

## Reporting

See [SECURITY.md](../SECURITY.md) for how to report vulnerabilities privately.
