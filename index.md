---
layout: page
title: k8s LogJedi
---

# k8s LogJedi

**May the logs be with you.** Your AI SRE Jedi that reads failures, writes fixes.

An AI-native Kubernetes sidekick that watches your pods, reads the logs, and turns failures into clear fixes—before they become outages.

## Quick links

- **[README](README.md)** — Full project overview, quick start (Helm & Kind), config, runbook, and API
- **[Requirements](docs/requirements.md)** — User stories and acceptance criteria
- **[API contract](docs/api-contract.md)** — LLM service and operator API
- **[Security](docs/security.md)** — Security considerations
- **[Release process](docs/release.md)** — Versioning and release
- **[ADRs](docs/adr/)** — Architecture decision records

## Install

**Helm:**

```bash
helm install logjedi ./charts/logjedi -n logjedi --create-namespace
```

**Kind (local):**

```bash
kind create cluster --name logjedi
make docker-build
kind load docker-image logjedi-llm-service:latest --name logjedi
kind load docker-image logjedi-operator:latest --name logjedi
kubectl apply -f llm-service/deploy/
kubectl apply -f operator/config/deploy/
```

See the [README](README.md) for configuration and full steps.
