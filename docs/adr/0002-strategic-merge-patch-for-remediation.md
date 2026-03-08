# ADR 0002: Use strategic merge patch for remediation

## Status

Accepted.

## Context

The LLM can suggest changes to a Deployment, Job, or Pod (e.g. image, env, replicas, resources). We need a safe way to apply these changes that respects Kubernetes semantics (e.g. list merge for containers) and limits blast radius.

## Decision

- Use **strategic merge patch** (`application/strategic-merge-patch+json`) as the only supported patch type for auto-apply.
- The operator **filters** the patch before applying: only allow `spec.replicas`, `spec.template.spec.containers[].name`, `image`, `env`, `resources`. Other fields are stripped so the LLM cannot change command, securityContext, or volumes in an uncontrolled way.
- Apply only when the patch **target** (kind, namespace, name) matches the resource that was analyzed and (when configured) the namespace is in the auto-apply allow list.
- Optionally run a **server-side dry-run** before applying so invalid or immutable changes are rejected without mutating the cluster.

## Consequences

- **Pros:** Strategic merge patch is well-defined for built-in types; container list merge works as expected; filtering keeps changes minimal and auditable; dry-run adds a safety check.
- **Cons:** Filter allowlist must be updated if we want to allow more fields; strategic merge can be surprising for complex nested structures; we do not support JSON patch or other patch types for now.

We may extend the allowlist in the future (e.g. liveness probe) via configuration or code change, with documentation and tests.
