package logbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPBackend fetches historical logs from a generic HTTP endpoint (e.g. Loki,
// Promtail, Fluentd, or any custom service that returns log lines).
// LOG_BACKEND_URL can include path; we append query params: namespace, pod, since_minutes.
// Response: JSON array of strings, or plain text (one line per line).
// Optional auth: set LOG_BACKEND_HEADER_* env in the caller and pass headers to NewHTTPBackend.
type HTTPBackend struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewHTTPBackend returns a LogBackend that GETs the baseURL with query params.
func NewHTTPBackend(baseURL string, httpClient *http.Client, headers map[string]string) *HTTPBackend {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPBackend{baseURL: baseURL, httpClient: httpClient, headers: headers}
}

// GetHistoricalLogs performs a GET request with namespace, pod, since_minutes
// and parses the response as JSON array of strings or newline-delimited text.
func (h *HTTPBackend) GetHistoricalLogs(ctx context.Context, namespace, podName string, since time.Duration) ([]string, error) {
	u, err := url.Parse(h.baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("namespace", namespace)
	q.Set("pod", podName)
	q.Set("since_minutes", fmt.Sprintf("%.0f", since.Minutes()))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("log backend returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Try JSON array first
	var arr []string
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}
	// Plain text: one line per line
	return parseLinesPlain(string(body)), nil
}

// LokiStyleQuery builds a minimal Loki-style query for documentation.
// Example: {namespace="default", pod="web"} with time range.
// Actual Loki API is POST /loki/api/v1/query_range with a different shape;
// this is a placeholder for a custom adapter that translates our GET params.
func LokiStyleQuery(namespace, pod string, since time.Duration) string {
	return fmt.Sprintf(`{namespace="%s", pod="%s"} (last %s)`, namespace, pod, since.Round(time.Minute))
}

func parseLinesPlain(body string) []string {
	return strings.Split(strings.TrimSuffix(body, "\n"), "\n")
}
