package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends messages to Slack via incoming webhook URL.
type SlackNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewSlackNotifier creates a notifier that POSTs to the Slack webhook.
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// SendIssue posts a formatted message to Slack (markdown-style in text block).
func (s *SlackNotifier) SendIssue(ctx context.Context, issue IssuePayload) error {
	text := fmt.Sprintf(
		"*LogSage: %s %s/%s*\n"+
			"Reason: %s\n"+
			"*Summary:* %s\n"+
			"*Root cause:* %s\n"+
			"*Recommendation:* %s\n"+
			"*Proposed patch (suggested, not yet applied):*\n```\n%s\n```",
		issue.ResourceKind, issue.Namespace, issue.ResourceName,
		issue.Reason,
		issue.Summary,
		issue.RootCause,
		issue.Recommendation,
		issue.PatchJSON,
	)
	body := map[string]interface{}{"text": text}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
