package notifier

import (
	"context"
	"fmt"
)

// PagerDutyNotifier sends issues to PagerDuty (placeholder).
// To implement: use PagerDuty Events API v2 (trigger event with summary, details, severity).
// Config: PAGERDUTY_ROUTING_KEY or PAGERDUTY_INTEGRATION_KEY.
type PagerDutyNotifier struct {
	routingKey string
}

// NewPagerDutyNotifier creates a notifier that will send to PagerDuty when implemented.
func NewPagerDutyNotifier(routingKey string) *PagerDutyNotifier {
	return &PagerDutyNotifier{routingKey: routingKey}
}

// SendIssue sends the issue to PagerDuty. Placeholder: returns error so callers know it is not implemented.
func (p *PagerDutyNotifier) SendIssue(ctx context.Context, issue IssuePayload) error {
	if p.routingKey == "" {
		return fmt.Errorf("pagerduty: routing key not configured")
	}
	_ = issue
	_ = ctx
	return fmt.Errorf("pagerduty notifier not yet implemented: use Slack or Teams for now")
}
