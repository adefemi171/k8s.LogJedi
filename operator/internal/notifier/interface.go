package notifier

import "context"

// IssuePayload contains the data sent to Slack/Teams.
type IssuePayload struct {
	ResourceKind   string
	ResourceName   string
	Namespace      string
	Reason         string
	Summary        string
	RootCause      string
	Recommendation string
	PatchJSON      string
}

// Notifier sends issue notifications (Slack, Teams).
type Notifier interface {
	SendIssue(ctx context.Context, issue IssuePayload) error
}
