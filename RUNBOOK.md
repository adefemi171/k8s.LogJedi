# LogSage Runbook

LogSage is an AI-native Kubernetes sidekick that watches your pods, reads the logs, and turns failures into clear fixes—before they become outages. This runbook gives short procedures for common operational issues. See also [README.md](README.md) and [ROADMAP.md](ROADMAP.md).

---

## Operator not reconciling

**Symptoms:** Failed workloads (e.g. CrashLoopBackOff) are not triggering analysis; no logs from the operator about "LLM analysis received".

**Checks:**

1. **Operator running:** `kubectl -n logsage get pods -l app=logsage-operator` — pod should be Running.
2. **RBAC:** Operator needs get/list on pods, pods/log, events; get/list/patch on deployments, jobs. Check ClusterRole and ClusterRoleBinding: `kubectl get clusterrole logsage-operator -o yaml`.
3. **Namespace filter:** If `WATCH_NAMESPACES` or `EXCLUDE_NAMESPACES` is set, the failing resource’s namespace might be excluded. Check ConfigMap `logsage-operator-config` and env vars.
4. **Cooldown:** If the same resource was analyzed recently, the operator skips until `ANALYZE_COOLDOWN_MINUTES` has passed. Check operator logs for "RequeueAfter" or wait for cooldown to expire.
5. **Logs:** `kubectl -n logsage logs -l app=logsage-operator --tail=200` and look for errors (e.g. "LLM analyze failed", "list events", "get pod logs").

**Actions:** Fix RBAC or namespace config; ensure LLM service is reachable (see below); restart operator if needed: `kubectl -n logsage rollout restart deployment/logsage-operator`.

---

## LLM returns no action

**Symptoms:** Operator logs "LLM analysis received" but no patch is applied and no notification contains a suggested patch.

**Checks:**

1. **Mock provider:** With `LLM_PROVIDER=mock`, the LLM service returns a fixed response; for Deployments it includes a sample action. For Pods/Jobs the mock may set `action: null`. So "no action" can be expected for some resources when using mock.
2. **Real LLM:** If using a real provider, the model might not output a valid structured action (e.g. missing or malformed patch). Check LLM service logs for errors or validation failures.
3. **APPLY_MODE:** In `manual` mode the operator does not apply; it only notifies. So "no action" in the message might mean the notification payload has an empty patch — check Slack/Teams or the logged `kubectl patch` command.

**Actions:** For mock, this is expected when no patch is returned. For real LLM, improve prompts or add retries/fallback (see ROADMAP). Confirm APPLY_MODE and notification config.

---

## Slack / Teams not receiving messages

**Symptoms:** Operator runs and gets an analysis, but no message appears in Slack or Teams.

**Checks:**

1. **Webhook URL:** Ensure `SLACK_WEBHOOK_URL` or `TEAMS_WEBHOOK_URL` is set in the operator ConfigMap/env. `kubectl -n logsage get configmap logsage-operator-config -o yaml`.
2. **APPLY_MODE:** Notifications are sent in `manual` mode, or in `auto` mode after applying (if notifiers are configured). In `auto` with no notifiers, only audit logs are written.
3. **Errors in logs:** Look for "send notification failed" in operator logs. That usually means the webhook returned non-2xx or network error.
4. **Webhook validity:** Test the webhook manually (e.g. `curl -X POST -H 'Content-Type: application/json' -d '{"text":"test"}' <SLACK_WEBHOOK_URL>`). For Teams, ensure the URL is an incoming webhook URL.

**Actions:** Fix or rotate webhook URLs; confirm ConfigMap is mounted and env is set; restart operator after changing config.

---

## LLM service unreachable

**Symptoms:** Operator logs "LLM analyze failed" with connection refused, timeout, or 5xx.

**Checks:**

1. **Service and pod:** `kubectl -n logsage get svc llm-service` and `kubectl -n logsage get pods -l app=llm-service`. Pod should be Running; service should target the pod.
2. **URL:** Operator must use in-cluster URL when both run in the same cluster, e.g. `http://llm-service.logsage.svc.cluster.local:8000`. Check `LLM_SERVICE_URL` in operator deployment.
3. **Network policy:** If cluster uses network policies, allow traffic from operator pod(s) to LLM service on port 8000.
4. **Health:** From a pod in the cluster, `curl http://llm-service.logsage.svc.cluster.local:8000/health`. Should return `{"status":"ok"}`.

**Actions:** Fix service/deployment; correct LLM_SERVICE_URL; relax or add network policy; ensure LLM service has resources and is not OOMKilled (check `kubectl describe pod`).

---

## Patch apply failed (auto mode)

**Symptoms:** Operator logs "dry-run patch failed" or "apply patch to deployment/job/pod" with an error.

**Checks:**

1. **Dry-run:** If `DRY_RUN_BEFORE_APPLY=true`, the first Patch is server-side dry-run. If it fails, the real patch is not applied. Fix the patch (e.g. invalid field or immutable field).
2. **Scope:** Operator only applies when target namespace/name matches the failed resource and (if set) namespace is in `AUTO_APPLY_NAMESPACES`. Check config.
3. **Patch content:** LLM may suggest an invalid or immutable change. Check operator logs for the patch size and consider logging a redacted patch for debugging. Inspect the resource: `kubectl get deployment <name> -n <ns> -o yaml`.

**Actions:** Correct the patch (manually if needed); tighten prompt or allowlist so the LLM does not suggest invalid fields; disable dry-run only if you accept the risk.

---

## Escalation

If the issue is not covered here:

1. Collect operator and LLM service logs, and (if applicable) a sample of the failing resource and events.
2. Open an issue in the repo with the runbook section that best matches the symptom and what you’ve already checked.
3. For security-sensitive issues (e.g. accidental secret exposure), do not paste full specs or logs; describe the scenario and redact.
