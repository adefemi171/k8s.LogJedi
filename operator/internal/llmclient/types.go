package llmclient

// AnalyzeRequest is the JSON body sent to POST /analyze.
type AnalyzeRequest struct {
	ResourceKind   string        `json:"resource_kind"`
	ResourceName   string        `json:"resource_name"`
	Namespace      string        `json:"namespace"`
	Reason         string        `json:"reason"`
	Events         []EventItem   `json:"events"`
	Nodes          []NodeContext `json:"nodes,omitempty"`
	Spec           interface{}   `json:"spec"`
	RecentLogs     []string      `json:"recent_logs"`
	HistoricalLogs []string      `json:"historical_logs"`
}

type EventItem struct {
	Type           string `json:"type"`
	Reason         string `json:"reason"`
	Message        string `json:"message"`
	FirstTimestamp string `json:"firstTimestamp,omitempty"`
	LastTimestamp  string `json:"lastTimestamp,omitempty"`
}

type NodeContext struct {
	Name       string              `json:"name"`
	Conditions []NodeConditionItem `json:"conditions,omitempty"`
	Events     []EventItem         `json:"events,omitempty"`
}

type NodeConditionItem struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

// AnalyzeResponse is the JSON response from POST /analyze.
type AnalyzeResponse struct {
	Summary        string       `json:"summary"`
	RootCause      string       `json:"root_cause"`
	Recommendation string       `json:"recommendation"`
	Action         *PatchAction `json:"action,omitempty"`
}

type PatchAction struct {
	Type      string                 `json:"type"`
	Target    PatchTarget            `json:"target"`
	PatchType string                 `json:"patch_type"`
	Patch     map[string]interface{} `json:"patch"`
}

type PatchTarget struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
