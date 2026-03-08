package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TeamsNotifier sends messages to Microsoft Teams via incoming webhook URL.
type TeamsNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewTeamsNotifier creates a notifier that POSTs to the Teams webhook.
func NewTeamsNotifier(webhookURL string) *TeamsNotifier {
	return &TeamsNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// SendIssue posts a simple message to Teams (message card with title and text).
func (t *TeamsNotifier) SendIssue(ctx context.Context, issue IssuePayload) error {
	// Teams webhook accepts a message card. JSON encoding will escape the text.
	text := fmt.Sprintf(
		"**Reason:** %s\n\n**Summary:** %s\n\n**Root cause:** %s\n\n**Recommendation:** %s\n\n**Proposed patch (suggested, not yet applied):**\n```\n%s\n```",
		issue.Reason,
		issue.Summary,
		issue.RootCause,
		issue.Recommendation,
		issue.PatchJSON,
	)
	body := map[string]interface{}{
		"@type":    "MessageCard",
		"@context": "https://schema.org/extensions",
		"summary":  "k8s LogJedi: " + issue.ResourceKind + " " + issue.Namespace + "/" + issue.ResourceName,
		"title":    "k8s LogJedi: " + issue.ResourceKind + " " + issue.Namespace + "/" + issue.ResourceName,
		"text":     text,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams webhook returned %d", resp.StatusCode)
	}
	return nil
}
