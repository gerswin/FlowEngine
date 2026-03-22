package instance

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
)

// GuardContext contains all data available to guard functions.
type GuardContext struct {
	Instance *Instance
	Event    string
	ActorID  shared.ID
	Roles    []string // Actor's roles (from JWT/context)
}

// ActionContext contains all data available to action functions.
type ActionContext struct {
	Instance *Instance
	Event    string
	ActorID  shared.ID
	Params   map[string]interface{}
}

// GuardFunc is a function that evaluates a guard condition.
// Returns nil if the guard passes, or an error describing why it failed.
type GuardFunc func(ctx GuardContext, params map[string]interface{}) error

// ActionFunc is a function that executes a post-transition action.
type ActionFunc func(ctx ActionContext) error

// Engine executes guards and actions for workflow transitions.
type Engine struct {
	guards  map[string]GuardFunc
	actions map[string]ActionFunc
}

// NewEngine creates a new engine with all built-in guards and actions registered.
func NewEngine() *Engine {
	e := &Engine{
		guards:  make(map[string]GuardFunc),
		actions: make(map[string]ActionFunc),
	}
	e.registerBuiltinGuards()
	e.registerBuiltinActions()
	return e
}

// RegisterGuard registers a custom guard function.
func (e *Engine) RegisterGuard(name string, fn GuardFunc) {
	e.guards[name] = fn
}

// RegisterAction registers a custom action function.
func (e *Engine) RegisterAction(name string, fn ActionFunc) {
	e.actions[name] = fn
}

// EvaluateGuards runs all guards for an event. Returns error on first failure.
func (e *Engine) EvaluateGuards(ctx GuardContext, guards []workflow.GuardConfig) error {
	for _, g := range guards {
		fn, ok := e.guards[g.Type]
		if !ok {
			return shared.InvalidInputError(fmt.Sprintf("unknown guard type: %s", g.Type))
		}
		if err := fn(ctx, g.Params); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteActions runs all actions after a transition. Stops on first error.
func (e *Engine) ExecuteActions(ctx ActionContext, actions []workflow.ActionConfig) error {
	for _, a := range actions {
		fn, ok := e.actions[a.Type]
		if !ok {
			// Unknown actions are skipped (not critical)
			continue
		}
		actCtx := ActionContext{
			Instance: ctx.Instance,
			Event:    ctx.Event,
			ActorID:  ctx.ActorID,
			Params:   a.Params,
		}
		if err := fn(actCtx); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) registerBuiltinGuards() {
	// field_exists: checks if a field exists in instance data
	e.guards["field_exists"] = func(ctx GuardContext, params map[string]interface{}) error {
		field, _ := params["field"].(string)
		if field == "" {
			return shared.InvalidInputError("guard 'field_exists' requires 'field' param")
		}
		if !ctx.Instance.Data().Has(field) {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' does not exist", field))
		}
		return nil
	}

	// field_not_empty: checks field exists and is not empty string or nil
	e.guards["field_not_empty"] = func(ctx GuardContext, params map[string]interface{}) error {
		field, _ := params["field"].(string)
		if field == "" {
			return shared.InvalidInputError("guard 'field_not_empty' requires 'field' param")
		}
		val := ctx.Instance.Data().Get(field)
		if val == nil {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' is empty", field))
		}
		if str, ok := val.(string); ok && strings.TrimSpace(str) == "" {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' is empty", field))
		}
		return nil
	}

	// field_equals: checks if a field equals a specific value
	e.guards["field_equals"] = func(ctx GuardContext, params map[string]interface{}) error {
		field, _ := params["field"].(string)
		value := params["value"]
		if field == "" {
			return shared.InvalidInputError("guard 'field_equals' requires 'field' param")
		}
		actual := ctx.Instance.Data().Get(field)
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", value) {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' expected '%v', got '%v'", field, value, actual))
		}
		return nil
	}

	// field_matches: checks if a string field matches a regex pattern
	e.guards["field_matches"] = func(ctx GuardContext, params map[string]interface{}) error {
		field, _ := params["field"].(string)
		pattern, _ := params["pattern"].(string)
		if field == "" || pattern == "" {
			return shared.InvalidInputError("guard 'field_matches' requires 'field' and 'pattern' params")
		}
		val, ok := ctx.Instance.Data().GetString(field)
		if !ok {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' is not a string", field))
		}
		matched, err := regexp.MatchString(pattern, val)
		if err != nil {
			return shared.InvalidInputError(fmt.Sprintf("guard 'field_matches' invalid pattern: %s", pattern))
		}
		if !matched {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: field '%s' does not match pattern '%s'", field, pattern))
		}
		return nil
	}

	// has_role: checks if the actor has a specific role
	e.guards["has_role"] = func(ctx GuardContext, params map[string]interface{}) error {
		role, _ := params["role"].(string)
		if role == "" {
			return shared.InvalidInputError("guard 'has_role' requires 'role' param")
		}
		for _, r := range ctx.Roles {
			if r == role {
				return nil
			}
		}
		return shared.ForbiddenError(fmt.Sprintf("guard failed: actor does not have role '%s'", role))
	}

	// has_any_role: checks if the actor has at least one of the specified roles
	e.guards["has_any_role"] = func(ctx GuardContext, params map[string]interface{}) error {
		rolesRaw, _ := params["roles"].([]interface{})
		if len(rolesRaw) == 0 {
			return shared.InvalidInputError("guard 'has_any_role' requires 'roles' param")
		}
		for _, r := range ctx.Roles {
			for _, required := range rolesRaw {
				if r == fmt.Sprintf("%v", required) {
					return nil
				}
			}
		}
		return shared.ForbiddenError("guard failed: actor does not have any of the required roles")
	}

	// validate_required_fields: checks multiple fields exist in data
	e.guards["validate_required_fields"] = func(ctx GuardContext, params map[string]interface{}) error {
		fieldsRaw, _ := params["fields"].([]interface{})
		if len(fieldsRaw) == 0 {
			return shared.InvalidInputError("guard 'validate_required_fields' requires 'fields' param")
		}
		var missing []string
		for _, f := range fieldsRaw {
			field := fmt.Sprintf("%v", f)
			if !ctx.Instance.Data().Has(field) {
				missing = append(missing, field)
			}
		}
		if len(missing) > 0 {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: missing required fields: %s", strings.Join(missing, ", ")))
		}
		return nil
	}

	// is_assigned_to_actor: checks if the instance is assigned to the actor performing the transition
	e.guards["is_assigned_to_actor"] = func(ctx GuardContext, params map[string]interface{}) error {
		assignedTo := ctx.Instance.Data().Get("assigned_to")
		if assignedTo == nil {
			return shared.InvalidInputError("guard failed: instance is not assigned to anyone")
		}
		if fmt.Sprintf("%v", assignedTo) != ctx.ActorID.String() {
			return shared.ForbiddenError("guard failed: instance is not assigned to the current actor")
		}
		return nil
	}

	// is_not_assigned: checks that the instance is NOT assigned
	e.guards["is_not_assigned"] = func(ctx GuardContext, params map[string]interface{}) error {
		assignedTo := ctx.Instance.Data().Get("assigned_to")
		if assignedTo != nil && fmt.Sprintf("%v", assignedTo) != "" {
			return shared.InvalidInputError("guard failed: instance is already assigned")
		}
		return nil
	}

	// substate_equals: checks if current substate matches
	e.guards["substate_equals"] = func(ctx GuardContext, params map[string]interface{}) error {
		expected, _ := params["substate"].(string)
		if expected == "" {
			return shared.InvalidInputError("guard 'substate_equals' requires 'substate' param")
		}
		if !ctx.Instance.HasSubState() {
			return shared.InvalidInputError("guard failed: instance has no substate")
		}
		if ctx.Instance.CurrentSubState().ID() != expected {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: substate is '%s', expected '%s'", ctx.Instance.CurrentSubState().ID(), expected))
		}
		return nil
	}

	// instance_age_less_than: checks instance was created less than X duration ago
	e.guards["instance_age_less_than"] = func(ctx GuardContext, params map[string]interface{}) error {
		durationStr, _ := params["duration"].(string)
		if durationStr == "" {
			return shared.InvalidInputError("guard 'instance_age_less_than' requires 'duration' param")
		}
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			return shared.InvalidInputError(fmt.Sprintf("guard 'instance_age_less_than' invalid duration: %s", durationStr))
		}
		age := time.Since(ctx.Instance.CreatedAt().Time())
		if age >= d {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: instance age %s exceeds %s", age.Round(time.Second), d))
		}
		return nil
	}

	// instance_age_more_than: checks instance was created more than X duration ago
	e.guards["instance_age_more_than"] = func(ctx GuardContext, params map[string]interface{}) error {
		durationStr, _ := params["duration"].(string)
		if durationStr == "" {
			return shared.InvalidInputError("guard 'instance_age_more_than' requires 'duration' param")
		}
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			return shared.InvalidInputError(fmt.Sprintf("guard 'instance_age_more_than' invalid duration: %s", durationStr))
		}
		age := time.Since(ctx.Instance.CreatedAt().Time())
		if age < d {
			return shared.InvalidInputError(fmt.Sprintf("guard failed: instance age %s is less than %s", age.Round(time.Second), d))
		}
		return nil
	}

	// custom: always passes (placeholder for external custom guards)
	e.guards["custom"] = func(ctx GuardContext, params map[string]interface{}) error {
		return nil
	}
}

func (e *Engine) registerBuiltinActions() {
	// set_metadata: sets a key/value pair in instance data
	e.actions["set_metadata"] = func(ctx ActionContext) error {
		key, _ := ctx.Params["key"].(string)
		value := ctx.Params["value"]
		if key == "" {
			return nil
		}
		// Handle special value "$now" to set current timestamp
		if str, ok := value.(string); ok && str == "$now" {
			value = time.Now().UTC().Format(time.RFC3339)
		}
		ctx.Instance.UpdateData(key, value)
		return nil
	}

	// increment_field: increments a numeric field in instance data
	e.actions["increment_field"] = func(ctx ActionContext) error {
		field, _ := ctx.Params["field"].(string)
		if field == "" {
			return nil
		}
		current, _ := ctx.Instance.Data().GetInt(field)
		ctx.Instance.UpdateData(field, current+1)
		return nil
	}

	// assign_to_user: sets the assigned_to field
	e.actions["assign_to_user"] = func(ctx ActionContext) error {
		userID, _ := ctx.Params["user_id"].(string)
		if userID == "" {
			// Default: assign to the actor performing the transition
			userID = ctx.ActorID.String()
		}
		ctx.Instance.UpdateData("assigned_to", userID)
		return nil
	}

	// emit_event: records a custom domain event (stored in data for now)
	e.actions["emit_event"] = func(ctx ActionContext) error {
		eventName, _ := ctx.Params["event_name"].(string)
		if eventName == "" {
			return nil
		}
		ctx.Instance.UpdateData("last_emitted_event", eventName)
		return nil
	}

	// mark_as_approved: sets approval metadata
	e.actions["mark_as_approved"] = func(ctx ActionContext) error {
		ctx.Instance.UpdateData("approved", true)
		ctx.Instance.UpdateData("approved_at", time.Now().UTC().Format(time.RFC3339))
		ctx.Instance.UpdateData("approved_by", ctx.ActorID.String())
		return nil
	}

	// mark_as_completed: sets completion metadata
	e.actions["mark_as_completed"] = func(ctx ActionContext) error {
		ctx.Instance.UpdateData("completed", true)
		ctx.Instance.UpdateData("completed_at", time.Now().UTC().Format(time.RFC3339))
		return nil
	}

	// increment_rejection_count: increments rejection_count field
	e.actions["increment_rejection_count"] = func(ctx ActionContext) error {
		current, _ := ctx.Instance.Data().GetInt("rejection_count")
		ctx.Instance.UpdateData("rejection_count", current+1)
		return nil
	}

	// add_feedback_to_instance: appends feedback to a feedback_history array in data
	e.actions["add_feedback_to_instance"] = func(ctx ActionContext) error {
		feedback, _ := ctx.Params["feedback"].(string)
		if feedback == "" {
			return nil
		}
		ctx.Instance.UpdateData("last_feedback", feedback)
		return nil
	}

	// update_document_type: updates document type field
	e.actions["update_document_type"] = func(ctx ActionContext) error {
		docType, _ := ctx.Params["document_type"].(string)
		if docType != "" {
			ctx.Instance.UpdateData("document_type", docType)
		}
		return nil
	}

	// log_reclassification: logs reclassification event
	e.actions["log_reclassification"] = func(ctx ActionContext) error {
		ctx.Instance.UpdateData("reclassified", true)
		ctx.Instance.UpdateData("reclassified_at", time.Now().UTC().Format(time.RFC3339))
		ctx.Instance.UpdateData("reclassified_by", ctx.ActorID.String())
		return nil
	}
}
