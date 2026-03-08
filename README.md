# k8s LogJedi

![K8s LogJedi](images/k8s_jedi.png)

[![code size](https://img.shields.io/github/languages/code-size/adefemi171/k8s.LogJedi)](https://github.com/adefemi171/k8s.LogJedi)
[![build](https://img.shields.io/github/actions/workflow/status/adefemi171/k8s.LogJedi/ci.yml?branch=main&label=build)](https://github.com/adefemi171/k8s.LogJedi/actions)
[![release](https://img.shields.io/github/v/release/adefemi171/k8s.LogJedi)](https://github.com/adefemi171/k8s.LogJedi/releases)
[![docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue)](https://adefemi171.github.io/k8s.LogJedi/)
[![License](https://img.shields.io/github/license/adefemi171/k8s.LogJedi)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/adefemi171/k8s.LogJedi?directory=operator&label=Go)](https://go.dev/)
[![last commit](https://img.shields.io/github/last-commit/adefemi171/k8s.LogJedi)](https://github.com/adefemi171/k8s.LogJedi/commits/main)

**May the logs be with you.** Your AI SRE Jedi that reads failures, writes fixes.

An AI-native Kubernetes sidekick that watches your pods, reads the logs, and turns failures into clear fixes—before they become outages. A production-ready operator that detects failed workloads, collects logs and events, and talks to a Python FastAPI + Strands LLM service for analysis and optional remediation (Slack/Teams notification, auto or manual apply).

**Current features:** Multi-LLM support (mock, OpenAI, Bedrock, Gemini) with `.env` loading; health/ready and OpenAPI; per-resource cooldown and dry-run before apply; log and namespace filters; operator–LLM auth; audit logging; secret redaction; resource limits; runbook and ADRs. See [ROADMAP.md](ROADMAP.md) for the full list and planned work.

## Table of contents

- [Architecture](#architecture)
- [Quickstart with kind/minikube](#quickstart-with-kindminikube)
- [Testing and validation (full flow)](#testing-and-validation-full-flow)
- [Config options](#config-options)
- [Runbook](#runbook)
- [Architecture decisions](#architecture-decisions)
- [Helm chart](#helm-chart)
- [Repository layout](#repository-layout)
- [Build and deploy](#build-and-deploy)
- [LLM service API](#llm-service-api)
- [Log backend (HTTP / Loki)](#log-backend-http--loki)

## Architecture

- **Operator (Go):** controller-runtime; watches Pods, Deployments, Jobs; on failure collects Events + recent pod logs (+ optional historical logs via pluggable backend); redacts secrets; POSTs to LLM service; applies patch (auto) or notifies (manual).
- **LLM service (Python):** FastAPI `POST /analyze`; Strands agent with structured output; mock or config-driven LLM provider.
- **Log backend:** Pluggable (in-cluster K8s API or HTTP/Loki-style).
- **Notifications:** Slack and Microsoft Teams webhooks for manual review.

## Quickstart with kind/minikube

### 1. Start a local cluster

**kind:**
```bash
kind create cluster --name logjedi
kubectl cluster-info --context kind-logjedi
```

**minikube:**
```bash
minikube start
kubectl config use-context minikube
```

### 2. Build and load images (if using kind)

```bash
make docker-build
# For kind, load images into the cluster:
kind load docker-image logjedi-llm-service:latest --name logjedi
kind load docker-image logjedi-operator:latest --name logjedi
```

For minikube, you can use the same images with `eval $(minikube docker-env)` before `make docker-build`, or use a registry.

### 3. Deploy the LLM service first

```bash
kubectl apply -f llm-service/deploy/
```

Wait until the LLM service pod is ready:
```bash
kubectl -n logjedi get pods -l app=llm-service
```

### 4. Deploy the operator

```bash
kubectl apply -f operator/config/deploy/
# This applies: namespace, ServiceAccount, ConfigMap, RBAC, Deployment
```

Check the operator is running:
```bash
kubectl -n logjedi get pods -l app=logjedi-operator
kubectl -n logjedi logs -l app=logjedi-operator -f
```

### 5. Create a sample failing deployment

```bash
kubectl apply -f operator/config/samples/failing-deployment.yaml
```

This creates a Deployment in `default` with an invalid image (`example.com/nonexistent:latest`), which will trigger `ImagePullBackOff` or `ErrImagePull`.

### 6. Expected flow

1. The operator watches Pods/Deployments/Jobs. When it sees the failing deployment (or its pods), it:
   - Collects Kubernetes Events for that resource
   - Fetches recent pod logs from the API
   - Optionally fetches historical logs from the configured log backend
   - Redacts secrets from the spec and POSTs to `http://llm-service.logjedi.svc.cluster.local:8000/analyze`

2. The LLM service returns a JSON response with `summary`, `root_cause`, `recommendation`, and optionally an `action` (e.g. `k8s_patch`).

3. **If `APPLY_MODE=manual`** (default): the operator sends the summary and proposed patch to Slack/Teams (if webhooks are configured) and logs a suggested `kubectl patch` command. It does not apply the patch.

4. **If `APPLY_MODE=auto`**: the operator applies the patch to the Deployment/Job/Pod (with scope validation and allowed-field filtering).

To test without Slack/Teams, leave `SLACK_WEBHOOK_URL` and `TEAMS_WEBHOOK_URL` unset; the operator will still log the suggested patch command.

## Testing and validation (full flow)

Use this to get the application running and confirm the LLM analysis and (optional) auto-apply work end-to-end.

### 1. Cluster and images

- Start a cluster (e.g. `kind create cluster --name logjedi` or `minikube start`).
- Build and load images:
  - `make docker-build`
  - kind: `kind load docker-image logjedi-llm-service:latest --name logjedi` and same for `logjedi-operator:latest`.
  - minikube: `eval $(minikube docker-env)` then `make docker-build`.

### 2. Deploy LLM service

```bash
kubectl apply -f llm-service/deploy/
kubectl -n logjedi get pods -l app=llm-service   # wait until Ready
```

For a real LLM (e.g. OpenAI), create a Secret and point the deployment at it:

```bash
kubectl -n logjedi create secret generic logjedi-llm-openai --from-literal=LLM_API_KEY=sk-your-key
```

The LLM service deployment in `llm-service/deploy/deployment.yaml` already uses `LLM_PROVIDER=openai` and `valueFrom.secretKeyRef` for `LLM_API_KEY`. If you use mock, you can leave the secret missing and set `LLM_PROVIDER=mock` in the deployment env.

### 3. Deploy operator and config

```bash
kubectl apply -f operator/config/deploy/
```

This applies namespace, ServiceAccount, ConfigMap, RBAC, and operator Deployment. Check the operator ConfigMap (`operator/config/deploy/configmap.yaml`): set `APPLY_MODE=auto` and `AUTO_APPLY_NAMESPACES: "default"` so auto-apply only runs in `default` (not in `logjedi`).

Verify:

```bash
kubectl -n logjedi get pods -l app=logjedi-operator
kubectl -n logjedi logs -l app=logjedi-operator -f
```

### 4. Trigger a failure and watch the flow

Create a failing deployment in `default` (e.g. invalid image):

```bash
kubectl apply -f operator/config/samples/failing-deployment.yaml
```

- In operator logs you should see: the operator detecting the failure, calling the LLM, and either "LLM analysis received" and (if auto) "audit: applied patch" or a suggested `kubectl patch` command.
- Check pods in `default`: after auto-apply the deployment may be patched to a pullable image (e.g. `nginx:latest`) and pods can become Ready.

### 5. Hit the LLM service from your machine

Port-forward and call the APIs:

```bash
kubectl -n logjedi port-forward svc/llm-service 8000:8000
```

Then:

- **Health:** `curl http://localhost:8000/health` and `curl http://localhost:8000/ready`
- **Analyses (error/summary from LLM):** open http://localhost:8000/analyses or `curl http://localhost:8000/analyses`
- **Reports (log of every LLM step):** open http://localhost:8000/reports or `curl http://localhost:8000/reports`
- **OpenAPI:** http://localhost:8000/openapi.json

After at least one analyze call (from the operator), `/analyses` shows the analysis results; `/reports` shows the same incidents as a step-by-step log (LLM steps plus "Operator applied patch" when the operator has applied the fix).

### 6. Optional: Helm install

Instead of raw manifests, you can install with Helm for parameterized config (namespace, apply mode, log backend, auto-apply namespaces, etc.). See [Helm chart](#helm-chart) below.

## Config options

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_SERVICE_URL` | Base URL of the LLM service | `http://llm-service:8000` (in-cluster: `http://llm-service.logjedi.svc.cluster.local:8000`) |
| `LOG_BACKEND_TYPE` | `kubernetes`, `loki`, or `custom_http` | `kubernetes` |
| `LOG_BACKEND_URL` | URL for HTTP log backend (for Loki/custom_http) | - |
| `APPLY_MODE` | `auto` or `manual` | `manual` |
| `SLACK_WEBHOOK_URL` | Slack incoming webhook URL | - |
| `TEAMS_WEBHOOK_URL` | Microsoft Teams incoming webhook URL | - |
| `LOG_TAIL_LINES` | Number of recent pod log lines to fetch | 100 |
| `LLM_PROVIDER` (llm-service) | `mock`, `openai`, `bedrock`, or `gemini` | `mock` |
| `LLM_API_KEY` (llm-service) | API key (OpenAI, Gemini AI Studio) | - |
| `ANALYZE_COOLDOWN_MINUTES` (operator) | Minutes before re-analyzing same resource | 15 |
| `DRY_RUN_BEFORE_APPLY` (operator) | If true, server-side dry-run before applying patch | false |
| `MAX_RECENT_LOG_LINES` (operator) | Cap recent_logs sent to LLM (0 = no cap) | 0 |
| `MAX_HISTORICAL_LOG_LINES` (operator) | Cap historical_logs (0 = no cap) | 0 |
| `WATCH_NAMESPACES` (operator) | Comma-separated namespaces to watch (empty = all) | - |
| `EXCLUDE_NAMESPACES` (operator) | Comma-separated namespaces to exclude | - |
| `AUTO_APPLY_NAMESPACES` (operator) | When auto, only apply in these namespaces (empty = all) | - |
| `LLM_SERVICE_AUTH_HEADER` (operator) | Optional Authorization header for operator→LLM (e.g. Bearer token) | - |

ConfigMap `logjedi-operator-config` in namespace `logjedi` can override these for the operator; set env vars on the LLM service deployment for the Python service.

### LLM service: .env and providers

The LLM service loads `.env` from `llm-service/.env` first, then from the **repo root `.env`**, so you can keep a single root `.env` with e.g. `LLM_PROVIDER=openai` and `LLM_API_KEY=sk-...` for local dev. For in-cluster, set env (or mount a Secret) on the LLM service Deployment. Copy `llm-service/.env.example` to `llm-service/.env` or add to root `.env`:

- **mock** (default): no real LLM; returns a fixed analysis. No key required.
- **openai**: set `LLM_PROVIDER=openai` and `LLM_API_KEY=sk-...` in root `.env` or `llm-service/.env`. Optional: `OPENAI_MODEL_ID` (default `gpt-4o`). Install optional deps: `pip install 'strands-agents[openai]'` (or add to pyproject and run `uv sync`). Then run the LLM service (see below).
- **bedrock**: set `LLM_PROVIDER=bedrock`, `AWS_REGION`, and optionally `BEDROCK_MODEL_ID`. Uses AWS credentials from env or IAM role (e.g. on EKS). Bedrock is included in the base Strands package.
- **gemini**: set `LLM_PROVIDER=gemini`, and either `LLM_API_KEY` (AI Studio) or `GOOGLE_APPLICATION_CREDENTIALS` (Vertex). Optionally `GEMINI_MODEL_ID` (default `gemini-2.0-flash`). Install: `pip install 'strands-agents[gemini]'`.

**Running the LLM service with your OpenAI key:** With `LLM_PROVIDER=openai` and `LLM_API_KEY=sk-...` in your root `.env`, run the service so the operator can call it. **In-cluster:** Deploy the LLM service (`kubectl apply -f llm-service/deploy/`), then create a Secret with your key and set the Deployment to use it (e.g. `envFrom` or `env` with `valueFrom.secretKeyRef`), and set `LLM_PROVIDER=openai` in the Deployment env. **Local:** From repo root, `cd llm-service && uv sync && uv run uvicorn main:app --host 0.0.0.0 --port 8000`; the operator in the cluster must be able to reach this (e.g. use a tunnel or run the operator locally pointing at your host).

## Runbook

See [RUNBOOK.md](RUNBOOK.md) for troubleshooting: operator not reconciling, LLM returns no action, Slack/Teams not receiving messages, LLM service unreachable, patch apply failures.

## Architecture decisions

See [docs/adr/](docs/adr/) for architecture decision records (e.g. operator/LLM split, strategic merge patch).

## Helm chart

A Helm chart under `charts/logjedi/` installs the operator and LLM service with one release. You can override namespace, apply mode, log backend type, auto-apply namespaces, and other options without editing YAML.

**Install (create namespace if needed):**
```bash
helm install logjedi ./charts/logjedi -n logjedi --create-namespace
```

**Override values (examples):**
```bash
# Install into a custom namespace with auto-apply only in default
helm install logjedi ./charts/logjedi -n my-ns --create-namespace \
  --set operatorConfig.applyMode=auto \
  --set operatorConfig.autoApplyNamespaces=default

# Use Loki log backend and manual apply
helm install logjedi ./charts/logjedi -n logjedi --create-namespace \
  --set operatorConfig.logBackendType=loki \
  --set operatorConfig.logBackendURL=https://loki.example.com/query \
  --set operatorConfig.applyMode=manual

# LLM service: use existing secret or create from value (dev only)
helm install logjedi ./charts/logjedi -n logjedi --create-namespace \
  --set llmService.existingSecret=logjedi-llm-openai
# Or create secret from chart: --set llmService.createSecret=true --set llmService.apiKey=sk-...
```

**Key values (see `charts/logjedi/values.yaml`):**
| Value | Description | Default |
|-------|-------------|---------|
| `namespace` | Namespace name (used when installing with `-n`) | `logjedi` |
| `operatorConfig.applyMode` | `auto` or `manual` | `auto` |
| `operatorConfig.logBackendType` | `kubernetes`, `loki`, or `custom_http` | `kubernetes` |
| `operatorConfig.autoApplyNamespaces` | Comma-separated namespaces for auto-apply (empty = all) | `default` |
| `operatorConfig.watchNamespaces` | Comma-separated namespaces to watch (empty = all) | - |
| `operatorConfig.excludeNamespaces` | Comma-separated namespaces to exclude | - |
| `operatorConfig.analyzeCooldownMinutes` | Cooldown before re-analyzing same resource | `15` |
| `operatorConfig.dryRunBeforeApply` | Server-side dry-run before applying patch | `false` |
| `llmService.provider` | `mock`, `openai`, `bedrock`, or `gemini` | `openai` |
| `llmService.existingSecret` | Secret name for LLM API key | `logjedi-llm-openai` |
| `operator.image.repository`, `.tag` | Operator image | `logjedi-operator:latest` |
| `llmService.image.repository`, `.tag` | LLM service image | `logjedi-llm-service:latest` |

After install, create the API key secret if needed, then port-forward to the LLM service and open `/analyses` and `/reports` as in [Testing and validation](#testing-and-validation-full-flow).

## Repository layout

- `operator/` – Go controller (controller-runtime), `controllers/`, `internal/` (logbackend, llmclient, notifier, redact, patch), `config/deploy/`, `config/samples/`.
- `llm-service/` – FastAPI app, Pydantic models, Strands analyzer, `llm_client`, `routers/`, `services/`, Dockerfile, `deploy/`, `.env.example`.
- `charts/logjedi/` – Helm chart for operator + LLM service with configurable values.
- `docs/adr/` – Architecture decision records; `docs/requirements.md`, `docs/security.md`, `docs/api-contract.md`, `docs/release.md` – requirements, security, API contract, and release process.
- [CONTRIBUTING.md](CONTRIBUTING.md) – How to contribute, run tests, and submit PRs; [SECURITY.md](SECURITY.md) – vulnerability reporting; [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) – community guidelines; [LICENSE](LICENSE) – Apache-2.0; [CHANGELOG.md](CHANGELOG.md) – version history.
- `.github/workflows/` – CI (build + test on push/PR), build-and-push (GHCR), release, golangci-lint, semantic PR, and GitHub Pages; `.github/dependabot.yml` – dependency updates; `.github/PULL_REQUEST_TEMPLATE.md` – PR checklist.
- [RUNBOOK.md](RUNBOOK.md) – Operational runbook; [ROADMAP.md](ROADMAP.md) – current state and roadmap.

## Build and deploy

- **Prerequisites:** Go 1.21+ for the operator. For the LLM service: **Python 3.14** and **[uv](https://docs.astral.sh/uv/)** (e.g. `brew install uv`). From `llm-service/`, run `uv sync` to install deps and `uv run uvicorn main:app --reload` for local dev.
- `make build` – build operator binary and install llm-service Python deps via `uv sync`.
- `make docker-build` – build both Docker images (`logjedi-operator:latest`, `logjedi-llm-service:latest`).
- **GitHub Container Registry:** On push to `main`, the "Build and push images" workflow builds both images and pushes them to GHCR. Each image is tagged with a **date-time** (UTC, `YYYYMMDD-HHMMSS`) and `latest`. Pull with `docker pull ghcr.io/<owner>/logjedi-operator:<tag>` and `ghcr.io/<owner>/logjedi-llm-service:<tag>` (use the workflow run or Packages page for the exact tag).
- `make deploy` – apply `operator/config/deploy/` and `llm-service/deploy/`.
- `make test` – run operator tests (go vet, go test) and llm-service tests (pytest; use `uv sync --extra dev` in llm-service for test deps).
- `make dev` – build and docker-build; prints instructions for cluster, deploy, and (for kind) loading images.
- `make build-operator`, `make docker-build-operator`, `make deploy-operator` – operator only.

## LLM service API

- **Health:** `GET /health` (liveness), `GET /ready` (readiness).
- **Analyze:** `POST /analyze` – submit a failed workload; returns summary, root cause, recommendation, and optional patch action.
- **Analyses:** `GET /analyses?limit=50` – returns the most recent analyses (error/summary the LLM produced). Newest first.
- **Reports:** `GET /reports?limit=50` – step-by-step log of what the LLM and operator did: each report has a `steps` array (e.g. "Received request for Pod X (ErrImagePull)", "Root cause: ...", "Recommendation: ...", "Suggested patch: container 'app' image from example.com/nonexistent:latest → nginx:latest", and when the operator applies the patch, "Operator applied patch to Pod X in namespace default"). Use for a full audit trail.
- **Report outcome (operator callback):** `POST /report-outcome` – the operator calls this after applying a patch so the report can append an "Operator applied" step. Body: `{"resource_kind","resource_name","namespace","outcome":"applied"}`.
- **OpenAPI:** `GET /openapi.json` (FastAPI-generated schema for all routes).

## Log backend (HTTP / Loki)

When `LOG_BACKEND_TYPE` is `loki` or `custom_http` and `LOG_BACKEND_URL` is set, the operator GETs the URL with query params `namespace`, `pod`, and `since_minutes` to fetch historical log lines. The endpoint should return either a JSON array of strings or plain text (one line per line). For Loki, you can run a small adapter that translates these params into a Loki query and returns lines.
