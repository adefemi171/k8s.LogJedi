package controllers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	coretyped "k8s.io/client-go/kubernetes/typed/core/v1"
	"github.com/go-logr/logr"

	"github.com/adefemi171/k8s.LogJedi/operator/internal/config"
	"github.com/adefemi171/k8s.LogJedi/operator/internal/logbackend"
	"github.com/adefemi171/k8s.LogJedi/operator/internal/llmclient"
	"github.com/adefemi171/k8s.LogJedi/operator/internal/notifier"
	"github.com/adefemi171/k8s.LogJedi/operator/internal/patch"
	"github.com/adefemi171/k8s.LogJedi/operator/internal/redact"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupWorkloadReconciler registers the workload controller with the manager.
func SetupWorkloadReconciler(mgr manager.Manager, cfg *config.Config) error {
	restConfig := mgr.GetConfig()
	coreV1, err := coretyped.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create core v1 client: %w", err)
	}

	llm := llmclient.NewClient(
		cfg.LLMServiceURL,
		cfg.LLMClientTimeout,
		cfg.LLMClientMaxRetries,
		cfg.LLMServiceAuthHeader,
	)

	var lb logbackend.LogBackend
	switch cfg.LogBackendType {
	case "loki", "custom_http":
		if cfg.LogBackendURL != "" {
			lb = logbackend.NewHTTPBackend(cfg.LogBackendURL, nil, nil)
		}
	default:
		lb = logbackend.NewKubernetesBackend(coreV1)
	}

	var notifiers []notifier.Notifier
	if cfg.SlackWebhookURL != "" {
		notifiers = append(notifiers, notifier.NewSlackNotifier(cfg.SlackWebhookURL))
	}
	if cfg.TeamsWebhookURL != "" {
		notifiers = append(notifiers, notifier.NewTeamsNotifier(cfg.TeamsWebhookURL))
	}

	r := &WorkloadReconciler{
		Client:       mgr.GetClient(),
		Config:       cfg,
		CoreV1:       coreV1,
		LLMClient:    llm,
		LogBackend:   lb,
		Notifiers:    notifiers,
		cooldownCache: &cooldownCache{m: make(map[string]time.Time)},
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("workload").
		Watches(&corev1.Pod{}, &handler.EnqueueRequestForObject{}).
		Watches(&appsv1.Deployment{}, &handler.EnqueueRequestForObject{}).
		Watches(&batchv1.Job{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

// cooldownCache tracks last analyze time per resource to avoid thundering herd.
type cooldownCache struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func (c *cooldownCache) shouldSkip(key string, cooldown time.Duration) (skip bool, requeueAfter time.Duration) {
	if cooldown <= 0 {
		return false, 0
	}
	c.mu.Lock()
	last := c.m[key]
	c.mu.Unlock()
	if last.IsZero() {
		return false, 0
	}
	elapsed := time.Since(last)
	if elapsed < cooldown {
		return true, cooldown - elapsed
	}
	return false, 0
}

func (c *cooldownCache) set(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = time.Now()
}

func resourceKey(kind, namespace, name string) string { return namespace + "/" + kind + "/" + name }

// namespaceWatched returns true if the namespace should be reconciled.
func (r *WorkloadReconciler) namespaceWatched(ns string) bool {
	for _, e := range r.Config.ExcludeNamespaces {
		if e == ns {
			return false
		}
	}
	if len(r.Config.WatchNamespaces) == 0 {
		return true
	}
	for _, w := range r.Config.WatchNamespaces {
		if w == ns {
			return true
		}
	}
	return false
}

// canAutoApplyIn returns true if auto-apply is allowed in this namespace.
func (r *WorkloadReconciler) canAutoApplyIn(ns string) bool {
	if len(r.Config.AutoApplyNamespaces) == 0 {
		return true
	}
	for _, a := range r.Config.AutoApplyNamespaces {
		if a == ns {
			return true
		}
	}
	return false
}

// truncateLogs caps the slice to maxLines (if > 0).
func truncateLogs(lines []string, maxLines int) []string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	out := make([]string, maxLines+1)
	copy(out, lines[:maxLines])
	out[maxLines] = fmt.Sprintf("... (%d more lines truncated)", len(lines)-maxLines)
	return out
}

// stringsOrEmpty returns s if non-nil, otherwise a non-nil empty slice (so JSON marshal sends [] not null).
func stringsOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// WorkloadReconciler reconciles Pods, Deployments, and Jobs for failure detection.
type WorkloadReconciler struct {
	client.Client
	Config        *config.Config
	CoreV1        coretyped.CoreV1Interface
	LLMClient     *llmclient.Client
	LogBackend    logbackend.LogBackend
	Notifiers     []notifier.Notifier
	cooldownCache *cooldownCache
}

// Reconcile handles a reconciliation request.
func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Namespace filter: skip if not in watch list or in exclude list.
	if !r.namespaceWatched(req.Namespace) {
		return ctrl.Result{}, nil
	}

	// Try to fetch as Pod, then Deployment, then Job (same name can exist for different kinds).
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err == nil {
		return r.reconcilePod(ctx, &pod, logger)
	}

	var dep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &dep); err == nil {
		return r.reconcileDeployment(ctx, &dep, logger)
	}

	var job batchv1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err == nil {
		return r.reconcileJob(ctx, &job, logger)
	}

	// Object not found or deleted - ignore
	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) reconcilePod(ctx context.Context, pod *corev1.Pod, logger logr.Logger) (ctrl.Result, error) {
	ok, reason := IsPodFailed(pod)
	if !ok {
		return ctrl.Result{}, nil
	}
	key := resourceKey("Pod", pod.Namespace, pod.Name)
	if skip, requeueAfter := r.cooldownCache.shouldSkip(key, r.Config.CooldownDuration); skip {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	events, err := r.collectEvents(ctx, pod.Namespace, pod.Name, string(pod.UID))
	if err != nil {
		logger.Error(err, "failed to list events")
	}
	recentLogs, err := r.getPodLogs(ctx, pod.Namespace, pod.Name, r.Config.LogTailLines, nil)
	if err != nil {
		logger.Error(err, "failed to get pod logs")
	}

	specRedacted, err := redact.RedactSpec(pod.Spec)
	if err != nil {
		logger.Error(err, "redact spec")
		return ctrl.Result{}, err
	}

	var historicalLogs []string
	if r.LogBackend != nil {
		if h, err := r.LogBackend.GetHistoricalLogs(ctx, pod.Namespace, pod.Name, r.Config.HistoricalLogSince); err == nil {
			historicalLogs = truncateLogs(h, r.Config.MaxHistoricalLogLines)
		}
	}
	recentLogs = truncateLogs(recentLogs, r.Config.MaxRecentLogLines)
	llmReq := &llmclient.AnalyzeRequest{
		ResourceKind:    "Pod",
		ResourceName:    pod.Name,
		Namespace:      pod.Namespace,
		Reason:         reason,
		Events:         eventsToItems(events),
		Spec:           specRedacted,
		RecentLogs:     stringsOrEmpty(recentLogs),
		HistoricalLogs: stringsOrEmpty(historicalLogs),
	}

	resp, err := r.LLMClient.Analyze(ctx, llmReq)
	if err != nil {
		logger.Error(err, "LLM analyze failed")
		return ctrl.Result{}, err
	}
	r.cooldownCache.set(key)
	logger.Info("LLM analysis received", "summary", resp.Summary)
	r.applyOrNotify(ctx, "Pod", pod.Namespace, pod.Name, reason, resp, logger)
	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) reconcileDeployment(ctx context.Context, dep *appsv1.Deployment, logger logr.Logger) (ctrl.Result, error) {
	ok, reason := IsDeploymentFailed(dep)
	if !ok {
		return ctrl.Result{}, nil
	}
	key := resourceKey("Deployment", dep.Namespace, dep.Name)
	if skip, requeueAfter := r.cooldownCache.shouldSkip(key, r.Config.CooldownDuration); skip {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	events, err := r.collectEvents(ctx, dep.Namespace, dep.Name, string(dep.UID))
	if err != nil {
		logger.Error(err, "failed to list events")
	}

	var podList corev1.PodList
	err = r.List(ctx, &podList, client.InNamespace(dep.Namespace), client.MatchingLabels(dep.Spec.Selector.MatchLabels))
	if err != nil {
		logger.Error(err, "list pods for deployment")
		return ctrl.Result{}, err
	}

	var recentLogs, historicalLogs []string
	for i := range podList.Items {
		if i >= 5 {
			break
		}
		p := &podList.Items[i]
		logs, _ := r.getPodLogs(ctx, p.Namespace, p.Name, r.Config.LogTailLines, nil)
		recentLogs = append(recentLogs, fmt.Sprintf("--- pod/%s ---", p.Name))
		recentLogs = append(recentLogs, logs...)
		if r.LogBackend != nil {
			if h, err := r.LogBackend.GetHistoricalLogs(ctx, p.Namespace, p.Name, r.Config.HistoricalLogSince); err == nil {
				historicalLogs = append(historicalLogs, fmt.Sprintf("--- pod/%s (historical) ---", p.Name))
				historicalLogs = append(historicalLogs, truncateLogs(h, r.Config.MaxHistoricalLogLines)...)
			}
		}
	}
	recentLogs = truncateLogs(recentLogs, r.Config.MaxRecentLogLines)

	specRedacted, err := redact.RedactSpec(dep.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	llmReq := &llmclient.AnalyzeRequest{
		ResourceKind:    "Deployment",
		ResourceName:    dep.Name,
		Namespace:      dep.Namespace,
		Reason:         reason,
		Events:         eventsToItems(events),
		Spec:           specRedacted,
		RecentLogs:     stringsOrEmpty(recentLogs),
		HistoricalLogs: stringsOrEmpty(historicalLogs),
	}

	resp, err := r.LLMClient.Analyze(ctx, llmReq)
	if err != nil {
		logger.Error(err, "LLM analyze failed")
		return ctrl.Result{}, err
	}
	r.cooldownCache.set(key)
	logger.Info("LLM analysis received", "summary", resp.Summary)
	r.applyOrNotify(ctx, "Deployment", dep.Namespace, dep.Name, reason, resp, logger)
	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) reconcileJob(ctx context.Context, job *batchv1.Job, logger logr.Logger) (ctrl.Result, error) {
	ok, reason := IsJobFailed(job)
	if !ok {
		return ctrl.Result{}, nil
	}
	key := resourceKey("Job", job.Namespace, job.Name)
	if skip, requeueAfter := r.cooldownCache.shouldSkip(key, r.Config.CooldownDuration); skip {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	events, err := r.collectEvents(ctx, job.Namespace, job.Name, string(job.UID))
	if err != nil {
		logger.Error(err, "failed to list events")
	}

	var podList corev1.PodList
	err = r.List(ctx, &podList, client.InNamespace(job.Namespace), client.MatchingLabels{"job-name": job.Name})
	if err != nil {
		logger.Error(err, "list pods for job")
		return ctrl.Result{}, err
	}

	var recentLogs, historicalLogs []string
	for i := range podList.Items {
		if i >= 5 {
			break
		}
		p := &podList.Items[i]
		logs, _ := r.getPodLogs(ctx, p.Namespace, p.Name, r.Config.LogTailLines, nil)
		recentLogs = append(recentLogs, fmt.Sprintf("--- pod/%s ---", p.Name))
		recentLogs = append(recentLogs, logs...)
		if r.LogBackend != nil {
			if h, err := r.LogBackend.GetHistoricalLogs(ctx, p.Namespace, p.Name, r.Config.HistoricalLogSince); err == nil {
				historicalLogs = append(historicalLogs, fmt.Sprintf("--- pod/%s (historical) ---", p.Name))
				historicalLogs = append(historicalLogs, truncateLogs(h, r.Config.MaxHistoricalLogLines)...)
			}
		}
	}
	recentLogs = truncateLogs(recentLogs, r.Config.MaxRecentLogLines)

	specRedacted, err := redact.RedactSpec(job.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}

	llmReq := &llmclient.AnalyzeRequest{
		ResourceKind:    "Job",
		ResourceName:    job.Name,
		Namespace:      job.Namespace,
		Reason:         reason,
		Events:         eventsToItems(events),
		Spec:           specRedacted,
		RecentLogs:     stringsOrEmpty(recentLogs),
		HistoricalLogs: stringsOrEmpty(historicalLogs),
	}

	resp, err := r.LLMClient.Analyze(ctx, llmReq)
	if err != nil {
		logger.Error(err, "LLM analyze failed")
		return ctrl.Result{}, err
	}
	r.cooldownCache.set(key)
	logger.Info("LLM analysis received", "summary", resp.Summary)
	r.applyOrNotify(ctx, "Job", job.Namespace, job.Name, reason, resp, logger)
	return ctrl.Result{}, nil
}

// applyOrNotify applies the patch when APPLY_MODE=auto, or notifies and logs when manual.
func (r *WorkloadReconciler) applyOrNotify(ctx context.Context, resourceKind, namespace, name, reason string, resp *llmclient.AnalyzeResponse, logger logr.Logger) {
	patchJSON := ""
	if resp.Action != nil {
		raw, _ := json.MarshalIndent(resp.Action.Patch, "", "  ")
		patchJSON = string(raw)
	}

	if r.Config.ApplyMode == "auto" && resp.Action != nil && resp.Action.Type == "k8s_patch" {
		t := resp.Action.Target
		if t.Namespace != namespace || t.Name != name {
			logger.Info("patch target namespace/name mismatch, skipping apply", "target", t.Namespace+"/"+t.Name)
			return
		}
		if !r.canAutoApplyIn(t.Namespace) {
			logger.Info("auto-apply not allowed in namespace, skipping", "namespace", t.Namespace)
			return
		}
		filtered := patch.FilterPatch(resp.Action.Patch)
		body, err := patch.ToJSON(filtered)
		if err != nil {
			logger.Error(err, "serialize patch")
			return
		}
		key := client.ObjectKey{Namespace: t.Namespace, Name: t.Name}
		switch t.Kind {
		case "Deployment":
			var d appsv1.Deployment
			if err := r.Get(ctx, key, &d); err != nil {
				logger.Error(err, "get deployment for patch")
				return
			}
			if r.Config.DryRunBeforeApply {
				if err := r.Patch(ctx, &d, client.RawPatch(types.StrategicMergePatchType, body), client.DryRunAll); err != nil {
					logger.Error(err, "dry-run patch failed for deployment, skipping apply")
					return
				}
			}
			if err := r.Patch(ctx, &d, client.RawPatch(types.StrategicMergePatchType, body)); err != nil {
				logger.Error(err, "apply patch to deployment")
				return
			}
			logger.Info("audit: applied patch to deployment", "kind", "Deployment", "namespace", t.Namespace, "name", t.Name, "patchSize", len(body))
			_ = r.LLMClient.ReportOutcome(ctx, &llmclient.ReportOutcomeRequest{ResourceKind: t.Kind, ResourceName: t.Name, Namespace: t.Namespace, Outcome: "applied"})
		case "Job":
			var j batchv1.Job
			if err := r.Get(ctx, key, &j); err != nil {
				logger.Error(err, "get job for patch")
				return
			}
			if r.Config.DryRunBeforeApply {
				if err := r.Patch(ctx, &j, client.RawPatch(types.StrategicMergePatchType, body), client.DryRunAll); err != nil {
					logger.Error(err, "dry-run patch failed for job, skipping apply")
					return
				}
			}
			if err := r.Patch(ctx, &j, client.RawPatch(types.StrategicMergePatchType, body)); err != nil {
				logger.Error(err, "apply patch to job")
				return
			}
			logger.Info("audit: applied patch to job", "kind", "Job", "namespace", t.Namespace, "name", t.Name, "patchSize", len(body))
			_ = r.LLMClient.ReportOutcome(ctx, &llmclient.ReportOutcomeRequest{ResourceKind: t.Kind, ResourceName: t.Name, Namespace: t.Namespace, Outcome: "applied"})
		case "Pod":
			var p corev1.Pod
			if err := r.Get(ctx, key, &p); err != nil {
				logger.Error(err, "get pod for patch")
				return
			}
			if r.Config.DryRunBeforeApply {
				if err := r.Patch(ctx, &p, client.RawPatch(types.StrategicMergePatchType, body), client.DryRunAll); err != nil {
					logger.Error(err, "dry-run patch failed for pod, skipping apply")
					return
				}
			}
			if err := r.Patch(ctx, &p, client.RawPatch(types.StrategicMergePatchType, body)); err != nil {
				logger.Error(err, "apply patch to pod")
				return
			}
			logger.Info("audit: applied patch to pod", "kind", "Pod", "namespace", t.Namespace, "name", t.Name, "patchSize", len(body))
			_ = r.LLMClient.ReportOutcome(ctx, &llmclient.ReportOutcomeRequest{ResourceKind: t.Kind, ResourceName: t.Name, Namespace: t.Namespace, Outcome: "applied"})
		default:
			logger.Info("unsupported patch target kind", "kind", t.Kind)
		}
		return
	}

	// Manual: notify and log kubectl command
	issue := notifier.IssuePayload{
		ResourceKind:   resourceKind,
		ResourceName:   name,
		Namespace:      namespace,
		Reason:         reason,
		Summary:        resp.Summary,
		RootCause:      resp.RootCause,
		Recommendation: resp.Recommendation,
		PatchJSON:      patchJSON,
	}
	for _, n := range r.Notifiers {
		if err := n.SendIssue(ctx, issue); err != nil {
			logger.Error(err, "send notification failed")
		}
	}
	if resp.Action != nil && resp.Action.Type == "k8s_patch" {
		t := resp.Action.Target
		logger.Info("manual apply: run kubectl patch",
			"command", fmt.Sprintf("kubectl patch %s %s -n %s -p '%s'", t.Kind, t.Name, t.Namespace, patchJSON),
		)
	}
}

func (r *WorkloadReconciler) collectEvents(ctx context.Context, namespace, name, uid string) ([]corev1.Event, error) {
	list, err := r.CoreV1.Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + name + ",involvedObject.namespace=" + namespace,
	})
	if err != nil {
		return nil, err
	}
	// Optionally filter by UID for precision
	out := make([]corev1.Event, 0, len(list.Items))
	for i := range list.Items {
		if list.Items[i].InvolvedObject.UID == types.UID(uid) {
			out = append(out, list.Items[i])
		}
	}
	if len(out) == 0 {
		out = list.Items
	}
	return out, nil
}

func (r *WorkloadReconciler) getPodLogs(ctx context.Context, namespace, podName string, tailLines int, sinceTime *metav1.Time) ([]string, error) {
	opts := &corev1.PodLogOptions{
		TailLines: ptr(int64(tailLines)),
	}
	if sinceTime != nil {
		opts.SinceTime = sinceTime
	}
	req := r.CoreV1.Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return readLines(stream)
}

func eventsToItems(events []corev1.Event) []llmclient.EventItem {
	out := make([]llmclient.EventItem, len(events))
	for i := range events {
		e := &events[i]
		out[i] = llmclient.EventItem{
			Type:           e.Type,
			Reason:         e.Reason,
			Message:        e.Message,
			FirstTimestamp: formatTime(e.FirstTimestamp),
			LastTimestamp:  formatTime(e.LastTimestamp),
		}
	}
	return out
}

func formatTime(t metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func readLines(stream io.Reader) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(stream)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func ptr[T any](v T) *T { return &v }
