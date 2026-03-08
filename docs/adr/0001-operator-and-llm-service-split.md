# ADR 0001: Operator and LLM service as separate components

## Status

Accepted.

## Context

We need to detect failed Kubernetes workloads, gather context (events, logs, spec), and use an LLM to produce a human-readable analysis and an optional machine-readable patch. The system must support multiple deployment environments (kind, minikube, cloud) and allow operators to choose LLM providers and approval flows.

## Decision

- **Operator (Go):** A single controller using `sigs.k8s.io/controller-runtime` watches Pods, Deployments, and Jobs. It detects failure signals, collects events and logs (in-cluster API + optional external log backend), redacts secrets, and sends a structured JSON request to an HTTP endpoint. It does not embed or call an LLM directly.
- **LLM service (Python):** A separate service (FastAPI) exposes `POST /analyze`. It receives the operator’s JSON, runs a Strands agent (or mock) to produce summary, root cause, recommendation, and optional patch, and returns JSON. It is stateless and can be scaled or swapped (e.g. different model or provider) without changing the operator.

Communication is HTTP/JSON only. The operator does not depend on Python or Strands; the LLM service does not depend on Kubernetes.

## Consequences

- **Pros:** Clear separation of concerns; operator stays small and fast; LLM stack can be upgraded or replaced independently; multiple clusters can share one LLM service; easier to test each component (operator with stub HTTP, LLM with curl).
- **Cons:** Operational overhead of two deployments; network and latency between operator and LLM; need to define and version the request/response schema and handle backward compatibility.

We accept the trade-off and document the API (OpenAPI, README) so that alternative LLM backends can be used without changing the operator.
