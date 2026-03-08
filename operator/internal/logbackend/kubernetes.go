package logbackend

import (
	"bufio"
	"context"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coretyped "k8s.io/client-go/kubernetes/typed/core/v1"
)

// KubernetesBackend fetches historical pod logs via the in-cluster Kubernetes API
// using PodLogOptions (SinceTime, TailLines).
type KubernetesBackend struct {
	coreV1 coretyped.CoreV1Interface
}

// NewKubernetesBackend returns a LogBackend that uses the cluster's Pod log API.
func NewKubernetesBackend(coreV1 coretyped.CoreV1Interface) *KubernetesBackend {
	return &KubernetesBackend{coreV1: coreV1}
}

// GetHistoricalLogs returns log lines for the pod from the last `since` period.
func (k *KubernetesBackend) GetHistoricalLogs(ctx context.Context, namespace, podName string, since time.Duration) ([]string, error) {
	sinceTime := metav1.NewTime(time.Now().Add(-since))
	opts := &corev1.PodLogOptions{
		SinceTime: &sinceTime,
		TailLines: ptr(int64(500)),
	}
	req := k.coreV1.Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return readStreamLines(stream)
}

func readStreamLines(r io.Reader) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func ptr[T any](v T) *T { return &v }
