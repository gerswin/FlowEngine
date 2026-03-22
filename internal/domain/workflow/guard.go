package workflow

// GuardConfig represents a guard definition from the workflow.
type GuardConfig struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// ActionConfig represents an action definition from the workflow.
type ActionConfig struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params,omitempty"`
}
