// Package yaml provides YAML parsing capabilities for workflow definitions.
package yaml

import (
	"time"
)

// WorkflowFile represents the root structure of a workflow YAML file.
type WorkflowFile struct {
	Version  string         `yaml:"version"`
	Workflow WorkflowSchema `yaml:"workflow"`
}

// WorkflowSchema represents the workflow definition in YAML.
type WorkflowSchema struct {
	ID           string          `yaml:"id"`
	Name         string          `yaml:"name"`
	Description  string          `yaml:"description"`
	InitialState string          `yaml:"initial_state"`
	States       []StateSchema   `yaml:"states"`
	Events       []EventSchema   `yaml:"events"`
	Webhooks     []WebhookSchema `yaml:"webhooks,omitempty"`
	SLA          *SLASchema      `yaml:"sla,omitempty"`
	Metadata     *MetadataSchema `yaml:"metadata,omitempty"`
}

// StateSchema represents a state definition in YAML.
type StateSchema struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description,omitempty"`
	Timeout      string            `yaml:"timeout,omitempty"`
	OnTimeout    string            `yaml:"on_timeout,omitempty"`
	Final        bool              `yaml:"final,omitempty"`
	AllowedRoles []string          `yaml:"allowed_roles,omitempty"`
	SubStates    []SubStateSchema  `yaml:"substates,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty"`
}

// SubStateSchema represents a substate within a state.
type SubStateSchema struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Default     bool   `yaml:"default,omitempty"`
}

// EventSchema represents an event (transition) definition in YAML.
type EventSchema struct {
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description,omitempty"`
	From           []string       `yaml:"from"`
	To             string         `yaml:"to"`
	FromSubstate   []string       `yaml:"from_substate,omitempty"`
	ToSubstate     string         `yaml:"to_substate,omitempty"`
	TargetSubstate string         `yaml:"target_substate,omitempty"`
	RequiredData   []string       `yaml:"required_data,omitempty"`
	Guards         []GuardSchema  `yaml:"guards,omitempty"`
	Actions        []ActionSchema `yaml:"actions,omitempty"`
}

// GuardSchema represents a guard condition in YAML.
type GuardSchema struct {
	Type   string                 `yaml:"type"`
	Params map[string]interface{} `yaml:"params,omitempty"`
}

// ActionSchema represents an action to execute during transition.
type ActionSchema struct {
	Type   string                 `yaml:"type"`
	Params map[string]interface{} `yaml:"params,omitempty"`
}

// WebhookSchema represents a webhook configuration.
type WebhookSchema struct {
	URL         string            `yaml:"url"`
	Events      []string          `yaml:"events"`
	Secret      string            `yaml:"secret,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	RetryConfig *RetryConfig      `yaml:"retry_config,omitempty"`
	Active      bool              `yaml:"active"`
}

// RetryConfig represents retry configuration for webhooks.
type RetryConfig struct {
	MaxRetries        int    `yaml:"max_retries"`
	BackoffMultiplier int    `yaml:"backoff_multiplier"`
	InitialInterval   string `yaml:"initial_interval"`
}

// SLASchema represents SLA configuration.
type SLASchema struct {
	OverallTarget      string                  `yaml:"overall_target"`
	StateTargets       map[string]string       `yaml:"state_targets,omitempty"`
	EscalationTriggers []EscalationTrigger     `yaml:"escalation_triggers,omitempty"`
}

// EscalationTrigger represents an escalation trigger configuration.
type EscalationTrigger struct {
	State     string `yaml:"state"`
	Threshold string `yaml:"threshold"`
	Action    string `yaml:"action"`
}

// MetadataSchema represents workflow metadata.
type MetadataSchema struct {
	Version    string            `yaml:"version,omitempty"`
	Author     string            `yaml:"author,omitempty"`
	CreatedAt  string            `yaml:"created_at,omitempty"`
	Tags       []string          `yaml:"tags,omitempty"`
	Category   string            `yaml:"category,omitempty"`
	Department string            `yaml:"department,omitempty"`
	Compliance []string          `yaml:"compliance,omitempty"`
	Extra      map[string]string `yaml:"extra,omitempty"`
}

// ParsedDuration represents a parsed duration string.
type ParsedDuration struct {
	Original string
	Duration time.Duration
}
