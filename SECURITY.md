# Security Policy

## Supported versions

| Version | Supported          |
|---------|--------------------|
| 0.1.x   | :white_check_mark: |
| < 0.1   | N/A                |

We support the latest minor release and may backport critical security fixes to the previous minor when practical.

## Reporting a vulnerability

**Please do not report security vulnerabilities in public GitHub issues.**

- **Email:** Open a [GitHub Security Advisory](https://github.com/adefemi171/k8s.LogJedi/security/advisories/new) (preferred) so the report is private, or contact the maintainers via the email listed in the repository if available.
- **What to include:** Description of the issue, steps to reproduce, impact, and suggested fix if you have one.
- **What to expect:** We will acknowledge and aim to respond within a reasonable time. We may ask for clarification. We will keep you updated on remediation and credit you in the advisory unless you prefer to stay anonymous.

## Security-related design

- **Secrets:** The operator redacts secret-like env vars and `valueFrom` (secretKeyRef/configMapKeyRef) from workload specs before sending them to the LLM. Do not rely on the LLM service as a trusted boundary for raw cluster secrets; keep API keys and tokens in Kubernetes Secrets or env and never commit `.env` (see [.gitignore](.gitignore)).
- **Operator–LLM auth:** Optional `LLM_SERVICE_AUTH_HEADER` (e.g. Bearer token) can be set so only the operator can call the LLM service. Use TLS in production.
- **RBAC:** The operator runs with a dedicated ServiceAccount and least-privilege ClusterRole (get/list/watch/patch on pods, deployments, jobs, events). Restrict `AUTO_APPLY_NAMESPACES` and use read-only log backends where possible.
- **Supply chain:** Dependencies are in `operator/go.mod` / `go.sum` and `llm-service/requirements.txt` (and pyproject.toml). We use Dependabot for dependency updates; review and test before merging.

For more detail, see [docs/security.md](docs/security.md).
