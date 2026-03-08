package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls the LLM service HTTP API with timeout and retries.
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	authHeader string // optional, e.g. "Bearer <token>" for Authorization header
}

// NewClient creates an LLM client. baseURL is the service base (e.g. http://llm-service:8000).
// authHeader is optional; when set it is sent as the Authorization header.
func NewClient(baseURL string, timeout time.Duration, maxRetries int, authHeader string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
		authHeader: authHeader,
	}
}

// Analyze sends the request to POST /analyze and returns the response.
func (c *Client) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/analyze"
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		if c.authHeader != "" {
			httpReq.Header.Set("Authorization", c.authHeader)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		var out AnalyzeResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &out, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", c.maxRetries+1, lastErr)
}

// ReportOutcomeRequest is the JSON body for POST /report-outcome.
type ReportOutcomeRequest struct {
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
	Outcome      string `json:"outcome"`
}

// ReportOutcome notifies the LLM service that the operator applied a patch (so it can append an "Operator applied" step to the report).
func (c *Client) ReportOutcome(ctx context.Context, req *ReportOutcomeRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal report-outcome request: %w", err)
	}
	url := c.baseURL + "/report-outcome"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authHeader != "" {
		httpReq.Header.Set("Authorization", c.authHeader)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("report-outcome status %d", resp.StatusCode)
	}
	return nil
}
