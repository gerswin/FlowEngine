package yaml

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// Parser parses workflow YAML files and creates domain Workflow objects.
type Parser struct {
	// allowedActions is a registry of valid action types
	allowedActions map[string]bool
	// allowedGuards is a registry of valid guard types
	allowedGuards map[string]bool
}

// NewParser creates a new YAML parser.
func NewParser() *Parser {
	return &Parser{
		allowedActions: defaultActions(),
		allowedGuards:  defaultGuards(),
	}
}

// defaultActions returns the default set of allowed actions.
func defaultActions() map[string]bool {
	return map[string]bool{
		"generate_document_id":       true,
		"set_metadata":               true,
		"emit_event":                 true,
		"assign_to_user":             true,
		"notify":                     true,
		"set_substate":               true,
		"mark_as_approved":           true,
		"send_to_recipient":          true,
		"mark_as_completed":          true,
		"create_escalation":          true,
		"record_escalation_response": true,
		"add_feedback_to_instance":   true,
		"increment_rejection_count":  true,
		"update_document_type":       true,
		"log_reclassification":       true,
		"create_clone":               true,
		"cancel_clone":               true,
		"cancel_escalation":          true,
		"increment_field":            true,
		"send_response":              true,
	}
}

// defaultGuards returns the default set of allowed guards.
func defaultGuards() map[string]bool {
	return map[string]bool{
		"has_role":                   true,
		"has_any_role":               true,
		"is_assigned_to_actor":       true,
		"is_not_assigned":            true,
		"validate_assignment":        true,
		"validate_required_fields":   true,
		"validate_escalation_exists": true,
		"can_reclassify":             true,
		"custom":                     true,
		"field_equals":               true,
		"field_not_empty":            true,
		"field_exists":               true,
		"field_matches":              true,
		"instance_age_less_than":     true,
		"instance_age_more_than":     true,
		"before_time":                true,
		"after_time":                 true,
		"on_weekday":                 true,
		"substate_equals":            true,
		"parent_state_equals":        true,
	}
}

// RegisterAction registers a custom action type.
func (p *Parser) RegisterAction(actionType string) {
	p.allowedActions[actionType] = true
}

// RegisterGuard registers a custom guard type.
func (p *Parser) RegisterGuard(guardType string) {
	p.allowedGuards[guardType] = true
}

// ParseFile parses a workflow from a YAML file path.
func (p *Parser) ParseFile(filepath string) (*workflow.Workflow, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.Parse(file)
}

// ParseBytes parses a workflow from YAML bytes.
func (p *Parser) ParseBytes(data []byte) (*workflow.Workflow, error) {
	reader := strings.NewReader(string(data))
	return p.Parse(reader)
}

// Parse parses a workflow from a YAML reader.
func (p *Parser) Parse(reader io.Reader) (*workflow.Workflow, error) {
	// Read all content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML content: %w", err)
	}

	// Parse YAML structure
	var workflowFile WorkflowFile
	if err := yaml.Unmarshal(data, &workflowFile); err != nil {
		return nil, InvalidYAMLError{Cause: err}
	}

	// Validate the parsed schema
	if err := p.validateSchema(&workflowFile); err != nil {
		return nil, err
	}

	// Convert to domain workflow
	return p.toDomainWorkflow(&workflowFile)
}

// validateSchema performs semantic validation on the parsed YAML.
func (p *Parser) validateSchema(wf *WorkflowFile) error {
	var errs ValidationErrors

	schema := &wf.Workflow

	// Validate version
	if wf.Version == "" {
		errs.Add("version", "file version is required")
	}

	// Validate workflow name
	if schema.Name == "" {
		errs.Add("workflow.name", "name is required")
	}

	// Validate initial state is specified
	if schema.InitialState == "" {
		errs.Add("workflow.initial_state", "initial_state is required")
	}

	// Validate states exist
	if len(schema.States) == 0 {
		errs.Add("workflow.states", "at least one state is required")
	}

	// Build state map and check for duplicates
	stateMap := make(map[string]bool)
	for i, state := range schema.States {
		if state.ID == "" {
			errs.Add(fmt.Sprintf("workflow.states[%d].id", i), "state ID is required")
			continue
		}

		if stateMap[state.ID] {
			errs.Add(fmt.Sprintf("workflow.states[%d].id", i), fmt.Sprintf("duplicate state ID: %s", state.ID))
			continue
		}
		stateMap[state.ID] = true

		// Validate state ID format
		if !isValidStateID(state.ID) {
			errs.Add(fmt.Sprintf("workflow.states[%d].id", i),
				fmt.Sprintf("invalid state ID format: %s (must match ^[a-z][a-z0-9_]*$)", state.ID))
		}

		// Validate state name
		if state.Name == "" {
			errs.Add(fmt.Sprintf("workflow.states[%d].name", i), "state name is required")
		}

		// Validate timeout duration format
		if state.Timeout != "" {
			if _, err := parseDuration(state.Timeout); err != nil {
				errs.Add(fmt.Sprintf("workflow.states[%d].timeout", i),
					fmt.Sprintf("invalid duration format: %s", state.Timeout))
			}

			// If timeout is set, on_timeout must be set
			if state.OnTimeout == "" {
				errs.Add(fmt.Sprintf("workflow.states[%d].on_timeout", i),
					"on_timeout event is required when timeout is specified")
			}
		}

		// Validate substates
		subStateMap := make(map[string]bool)
		for j, subState := range state.SubStates {
			if subState.ID == "" {
				errs.Add(fmt.Sprintf("workflow.states[%d].substates[%d].id", i, j), "substate ID is required")
				continue
			}
			if subStateMap[subState.ID] {
				errs.Add(fmt.Sprintf("workflow.states[%d].substates[%d].id", i, j),
					fmt.Sprintf("duplicate substate ID: %s", subState.ID))
			}
			subStateMap[subState.ID] = true
		}
	}

	// Validate initial state exists
	if schema.InitialState != "" && !stateMap[schema.InitialState] {
		errs.Add("workflow.initial_state",
			fmt.Sprintf("initial state '%s' is not defined in states list", schema.InitialState))
	}

	// Build event map and validate events
	eventMap := make(map[string]bool)
	for i, event := range schema.Events {
		if event.Name == "" {
			errs.Add(fmt.Sprintf("workflow.events[%d].name", i), "event name is required")
			continue
		}

		if eventMap[event.Name] {
			errs.Add(fmt.Sprintf("workflow.events[%d].name", i),
				fmt.Sprintf("duplicate event name: %s", event.Name))
			continue
		}
		eventMap[event.Name] = true

		// Validate destination state exists
		if event.To == "" {
			errs.Add(fmt.Sprintf("workflow.events[%d].to", i), "destination state is required")
		} else if !stateMap[event.To] {
			errs.Add(fmt.Sprintf("workflow.events[%d].to", i),
				fmt.Sprintf("destination state '%s' does not exist", event.To))
		}

		// Validate source states exist (empty from is allowed for initial events)
		for j, from := range event.From {
			if !stateMap[from] {
				errs.Add(fmt.Sprintf("workflow.events[%d].from[%d]", i, j),
					fmt.Sprintf("source state '%s' does not exist", from))
			}
		}

		// Validate guards
		for j, guard := range event.Guards {
			if guard.Type == "" {
				errs.Add(fmt.Sprintf("workflow.events[%d].guards[%d].type", i, j), "guard type is required")
			} else if !p.allowedGuards[guard.Type] {
				errs.Add(fmt.Sprintf("workflow.events[%d].guards[%d].type", i, j),
					fmt.Sprintf("unknown guard type: %s", guard.Type))
			}
		}

		// Validate actions
		for j, action := range event.Actions {
			if action.Type == "" {
				errs.Add(fmt.Sprintf("workflow.events[%d].actions[%d].type", i, j), "action type is required")
			} else if !p.allowedActions[action.Type] {
				errs.Add(fmt.Sprintf("workflow.events[%d].actions[%d].type", i, j),
					fmt.Sprintf("unknown action type: %s", action.Type))
			}
		}
	}

	// Validate on_timeout events exist
	for i, state := range schema.States {
		if state.OnTimeout != "" && !eventMap[state.OnTimeout] {
			errs.Add(fmt.Sprintf("workflow.states[%d].on_timeout", i),
				fmt.Sprintf("timeout event '%s' does not exist", state.OnTimeout))
		}
	}

	// Validate webhooks
	for i, webhook := range schema.Webhooks {
		if webhook.URL == "" {
			errs.Add(fmt.Sprintf("workflow.webhooks[%d].url", i), "webhook URL is required")
		}
		if len(webhook.Events) == 0 {
			errs.Add(fmt.Sprintf("workflow.webhooks[%d].events", i), "at least one event is required for webhook")
		}
		if webhook.RetryConfig != nil && webhook.RetryConfig.InitialInterval != "" {
			if _, err := parseDuration(webhook.RetryConfig.InitialInterval); err != nil {
				errs.Add(fmt.Sprintf("workflow.webhooks[%d].retry_config.initial_interval", i),
					fmt.Sprintf("invalid duration format: %s", webhook.RetryConfig.InitialInterval))
			}
		}
	}

	// Validate SLA durations
	if schema.SLA != nil {
		if schema.SLA.OverallTarget != "" {
			if _, err := parseDuration(schema.SLA.OverallTarget); err != nil {
				errs.Add("workflow.sla.overall_target",
					fmt.Sprintf("invalid duration format: %s", schema.SLA.OverallTarget))
			}
		}
		for stateName, duration := range schema.SLA.StateTargets {
			if _, err := parseDuration(duration); err != nil {
				errs.Add(fmt.Sprintf("workflow.sla.state_targets.%s", stateName),
					fmt.Sprintf("invalid duration format: %s", duration))
			}
		}
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}

// toDomainWorkflow converts a parsed YAML schema to a domain Workflow.
func (p *Parser) toDomainWorkflow(wf *WorkflowFile) (*workflow.Workflow, error) {
	schema := &wf.Workflow

	// Build state map first
	stateMap := make(map[string]workflow.State)
	for _, stateSchema := range schema.States {
		state, err := workflow.NewState(stateSchema.ID, stateSchema.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create state %s: %w", stateSchema.ID, err)
		}

		// Add description
		if stateSchema.Description != "" {
			state = state.WithDescription(stateSchema.Description)
		}

		// Add timeout
		if stateSchema.Timeout != "" {
			duration, _ := parseDuration(stateSchema.Timeout)
			state = state.WithTimeout(duration, stateSchema.OnTimeout)
		}

		// Mark as final
		if stateSchema.Final {
			state = state.AsFinal()
		}

		stateMap[stateSchema.ID] = state
	}

	// Get initial state
	initialState, exists := stateMap[schema.InitialState]
	if !exists {
		return nil, InitialStateNotFoundError{StateID: schema.InitialState}
	}

	// Create workflow with a system actor for YAML imports
	systemActor := shared.NewID()
	wfl, err := workflow.NewWorkflow(schema.Name, initialState, systemActor)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Set description
	if schema.Description != "" {
		wfl.SetDescription(schema.Description)
	}

	// Add remaining states (initial state is already added by NewWorkflow)
	for id, state := range stateMap {
		if id != schema.InitialState {
			if err := wfl.AddState(state); err != nil {
				return nil, fmt.Errorf("failed to add state %s: %w", id, err)
			}
		}
	}

	// Add events
	for _, eventSchema := range schema.Events {
		// Build source states
		var sources []workflow.State
		for _, fromID := range eventSchema.From {
			if state, exists := stateMap[fromID]; exists {
				sources = append(sources, state)
			}
		}

		// Handle initial events (empty from)
		if len(sources) == 0 {
			// For initial events, use initial state as source
			sources = []workflow.State{initialState}
		}

		// Get destination state
		destState, exists := stateMap[eventSchema.To]
		if !exists {
			return nil, StateNotFoundError{StateID: eventSchema.To, ReferencedBy: fmt.Sprintf("event %s", eventSchema.Name)}
		}

		// Create event
		event, err := workflow.NewEvent(eventSchema.Name, sources, destState)
		if err != nil {
			return nil, fmt.Errorf("failed to create event %s: %w", eventSchema.Name, err)
		}

		// Add validators from guards (backward compatibility)
		var validators []string
		for _, guard := range eventSchema.Guards {
			validators = append(validators, guard.Type)
		}
		if len(validators) > 0 {
			event = event.WithValidators(validators)
		}

		// Map guards to domain GuardConfig
		var guards []workflow.GuardConfig
		for _, g := range eventSchema.Guards {
			guards = append(guards, workflow.GuardConfig{Type: g.Type, Params: g.Params})
		}
		if len(guards) > 0 {
			event = event.WithGuards(guards)
		}

		// Map actions to domain ActionConfig
		var actions []workflow.ActionConfig
		for _, a := range eventSchema.Actions {
			actions = append(actions, workflow.ActionConfig{Type: a.Type, Params: a.Params})
		}
		if len(actions) > 0 {
			event = event.WithActions(actions)
		}

		// Add required data fields
		if len(eventSchema.RequiredData) > 0 {
			event = event.WithRequiredData(eventSchema.RequiredData)
		}

		// Add event to workflow
		if err := wfl.AddEvent(event); err != nil {
			return nil, fmt.Errorf("failed to add event %s: %w", eventSchema.Name, err)
		}
	}

	// Validate the final workflow
	if err := wfl.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	return wfl, nil
}

var stateIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// isValidStateID checks if a state ID matches the required pattern.
func isValidStateID(id string) bool {
	return stateIDPattern.MatchString(id)
}

// parseDuration parses a duration string (e.g., "24h", "30m", "2d").
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Handle days (Go's time.ParseDuration doesn't support days)
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid days format: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Use standard Go duration parsing for h, m, s, etc.
	return time.ParseDuration(s)
}

// ParseResult contains the result of parsing a YAML file.
type ParseResult struct {
	Workflow *workflow.Workflow
	Schema   *WorkflowFile
	Warnings []string
}

// ParseWithDetails parses a workflow and returns additional details.
func (p *Parser) ParseWithDetails(reader io.Reader) (*ParseResult, error) {
	// Read all content
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML content: %w", err)
	}

	// Parse YAML structure
	var workflowFile WorkflowFile
	if err := yaml.Unmarshal(data, &workflowFile); err != nil {
		return nil, InvalidYAMLError{Cause: err}
	}

	// Collect warnings
	var warnings []string

	// Check for unknown fields or deprecated features
	if workflowFile.Workflow.ID != "" {
		warnings = append(warnings, "workflow.id is ignored; ID will be generated automatically")
	}

	// Validate the parsed schema
	if err := p.validateSchema(&workflowFile); err != nil {
		return nil, err
	}

	// Convert to domain workflow
	wfl, err := p.toDomainWorkflow(&workflowFile)
	if err != nil {
		return nil, err
	}

	return &ParseResult{
		Workflow: wfl,
		Schema:   &workflowFile,
		Warnings: warnings,
	}, nil
}
