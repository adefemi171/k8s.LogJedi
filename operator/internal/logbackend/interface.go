package logbackend

import (
	"context"
	"time"
)

// LogBackend fetches historical logs for a pod. Recent logs are always
// obtained from the Kubernetes API by the operator.
type LogBackend interface {
	GetHistoricalLogs(ctx context.Context, namespace, podName string, since time.Duration) ([]string, error)
}
