package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	appInstance "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

// InstanceHandler handles HTTP requests for workflow instances.
type InstanceHandler struct {
	createUseCase     *appInstance.CreateInstanceUseCase
	getUseCase        *appInstance.GetInstanceUseCase
	transitionUseCase *appInstance.TransitionInstanceUseCase
	cloneUseCase      *appInstance.CloneInstanceUseCase
}

// NewInstanceHandler creates a new instance handler.
func NewInstanceHandler(
	createUseCase *appInstance.CreateInstanceUseCase,
	getUseCase *appInstance.GetInstanceUseCase,
	transitionUseCase *appInstance.TransitionInstanceUseCase,
	cloneUseCase *appInstance.CloneInstanceUseCase,
) *InstanceHandler {
	return &InstanceHandler{
		createUseCase:     createUseCase,
		getUseCase:        getUseCase,
		transitionUseCase: transitionUseCase,
		cloneUseCase:      cloneUseCase,
	}
}

// CreateInstanceRequest represents the request body for creating an instance.
type CreateInstanceRequest struct {
	Data struct {
		Type       string `json:"type" binding:"required"`
		Attributes struct {
			WorkflowID string                 `json:"workflow_id" binding:"required"`
			ParentID   string                 `json:"parent_id"` // Optional: for subprocesses
			StartedBy  string                 `json:"started_by" binding:"required"`
			Data       map[string]interface{} `json:"data"`
			Variables  map[string]interface{} `json:"variables"`
		} `json:"attributes" binding:"required"`
	} `json:"data" binding:"required"`
}

// TransitionRequest represents the request body for a transition.
type TransitionRequest struct {
	Data struct {
		Type       string `json:"type" binding:"required"`
		Attributes struct {
			Event    string                 `json:"event" binding:"required"`
			ActorID  string                 `json:"actor_id" binding:"required"`
			Reason   string                 `json:"reason"`
			Feedback string                 `json:"feedback"`
			Metadata map[string]interface{} `json:"metadata"`
			Data     map[string]interface{} `json:"data"`
		} `json:"attributes" binding:"required"`
	} `json:"data" binding:"required"`
}

// CreateInstance handles POST /api/v1/instances
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error()),
		})
		return
	}

	if req.Data.Type != "instance" {
		jsonapi.ErrorResponse(c, http.StatusConflict, []*jsonapi.Error{
			jsonapi.NewError(http.StatusConflict, "INVALID_TYPE", "Invalid resource type", "Expected type 'instance'"),
		})
		return
	}

	cmd := appInstance.CreateInstanceCommand{
		WorkflowID: req.Data.Attributes.WorkflowID,
		ParentID:   req.Data.Attributes.ParentID,
		StartedBy:  req.Data.Attributes.StartedBy,
		Data:       req.Data.Attributes.Data,
		Variables:  req.Data.Attributes.Variables,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	// Convert DTO to JSON:API Resource
	attributes := map[string]interface{}{
		"workflow_id":   result.WorkflowID,
		"current_state": result.CurrentState,
		"status":        result.Status,
		"version":       result.Version,
		"created_at":    result.CreatedAt,
		"updated_at":    result.UpdatedAt,
	}
	
	if result.ParentID != "" {
		attributes["parent_id"] = result.ParentID
	}

	resource := jsonapi.NewResource("instance", result.ID, attributes)

	jsonapi.SuccessResponse(c, http.StatusCreated, resource, nil, &jsonapi.Links{
		Self: "/api/v1/instances/" + result.ID,
	})
}

// CloneInstanceRequest represents the request body for cloning an instance.
type CloneInstanceRequest struct {
	Data struct {
		Type       string `json:"type" binding:"required"`
		Attributes struct {
			Assignees []struct {
				UserID   string `json:"user_id" binding:"required"`
				OfficeID string `json:"office_id"`
			} `json:"assignees" binding:"required"`
			ConsolidatorID  string                 `json:"consolidator_id" binding:"required"`
			Reason          string                 `json:"reason" binding:"required"`
			TimeoutDuration string                 `json:"timeout_duration" binding:"required"`
			Metadata        map[string]interface{} `json:"metadata"`
		} `json:"attributes" binding:"required"`
	} `json:"data" binding:"required"`
}

// CloneInstance handles POST /api/v1/instances/:id/clone
func (h *InstanceHandler) CloneInstance(c *gin.Context) {
	instanceID := c.Param("id")

	var req CloneInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error()),
		})
		return
	}

	if req.Data.Type != "clone-request" {
		jsonapi.ErrorResponse(c, http.StatusConflict, []*jsonapi.Error{
			jsonapi.NewError(http.StatusConflict, "INVALID_TYPE", "Invalid resource type", "Expected type 'clone-request'"),
		})
		return
	}

	assignees := make([]appInstance.CloneAssignee, len(req.Data.Attributes.Assignees))
	for i, a := range req.Data.Attributes.Assignees {
		assignees[i] = appInstance.CloneAssignee{
			UserID:   a.UserID,
			OfficeID: a.OfficeID,
		}
	}

	cmd := appInstance.CloneInstanceCommand{
		ParentInstanceID: instanceID,
		Assignees:        assignees,
		ConsolidatorID:   req.Data.Attributes.ConsolidatorID,
		Reason:           req.Data.Attributes.Reason,
		TimeoutDuration:  req.Data.Attributes.TimeoutDuration,
		Metadata:         req.Data.Attributes.Metadata,
	}

	result, err := h.cloneUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resource := jsonapi.NewResource("clone-group", result.CloneGroupID, map[string]interface{}{
		"parent_id":        result.ParentID,
		"cloned_instances": result.ClonedInstances,
		"expires_at":       result.ExpiresAt,
	})

	jsonapi.SuccessResponse(c, http.StatusCreated, resource, nil, &jsonapi.Links{
		Self: "/api/v1/instances/" + instanceID + "/clones",
	})
}

// GetInstance handles GET /api/v1/instances/:id
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	instanceID := c.Param("id")

	result, err := h.getUseCase.Execute(c.Request.Context(), instanceID)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resource := jsonapi.NewResource("instance", result.ID, map[string]interface{}{
		"workflow_id":   result.WorkflowID,
		"current_state": result.CurrentState,
		"status":        result.Status,
		"version":       result.Version,
		"created_at":    result.CreatedAt,
		"updated_at":    result.UpdatedAt,
		"data":          result.Data,
		"variables":     result.Variables,
	})

	jsonapi.SuccessResponse(c, http.StatusOK, resource, nil, &jsonapi.Links{
		Self: "/api/v1/instances/" + result.ID,
	})
}

// ListInstances handles GET /api/v1/instances
func (h *InstanceHandler) ListInstances(c *gin.Context) {
	queryOpts := jsonapi.ParseQueryOptions(c)

	// Support legacy query param or JSON:API filter
	workflowID := c.Query("workflow_id")
	if val, ok := queryOpts.Filters["workflow_id"]; ok {
		workflowID = val
	}

	result, err := h.getUseCase.ExecuteList(c.Request.Context(), queryOpts.PageNumber, queryOpts.PageSize, workflowID)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resources := make([]*jsonapi.Resource, len(result.Items))
	for i, res := range result.Items {
		resources[i] = jsonapi.NewResource("instance", res.ID, map[string]interface{}{
			"workflow_id":   res.WorkflowID,
			"current_state": res.CurrentState,
			"status":        res.Status,
			"version":       res.Version,
			"created_at":    res.CreatedAt,
			"updated_at":    res.UpdatedAt,
		})
		resources[i].Links = &jsonapi.Links{
			Self: "/api/v1/instances/" + res.ID,
		}
	}

	total := int(result.Total)
	totalPages := (total + queryOpts.PageSize - 1) / queryOpts.PageSize
	meta := map[string]interface{}{
		"total": total,
		"page": map[string]int{
			"number": queryOpts.PageNumber,
			"size":   queryOpts.PageSize,
			"total":  totalPages,
		},
	}

	baseLink := c.Request.URL.Path + "?"
	jsonapi.SuccessResponse(c, http.StatusOK, resources, meta, &jsonapi.Links{
		Self:  c.Request.URL.String(),
		First: baseLink + "page[number]=1&page[size]=" + strconv.Itoa(queryOpts.PageSize),
	})
}

// GetInstanceHistory handles GET /api/v1/instances/:id/history
func (h *InstanceHandler) GetInstanceHistory(c *gin.Context) {
	instanceID := c.Param("id")

	transitions, err := h.getUseCase.ExecuteHistory(c.Request.Context(), instanceID)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resources := make([]*jsonapi.Resource, len(transitions))
	for i, t := range transitions {
		resources[i] = jsonapi.NewResource("transition", t.ID, map[string]interface{}{
			"from":      t.From,
			"to":        t.To,
			"event":     t.Event,
			"actor":     t.Actor,
			"timestamp": t.Timestamp,
			"reason":    t.Reason,
			"feedback":  t.Feedback,
			"metadata":  t.Metadata,
		})
	}

	jsonapi.SuccessResponse(c, http.StatusOK, resources, nil, &jsonapi.Links{
		Self: "/api/v1/instances/" + instanceID + "/history",
	})
}

// TransitionInstance handles POST /api/v1/instances/:id/transitions
func (h *InstanceHandler) TransitionInstance(c *gin.Context) {
	instanceID := c.Param("id")

	var req TransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error()),
		})
		return
	}

	if req.Data.Type != "transition" {
		jsonapi.ErrorResponse(c, http.StatusConflict, []*jsonapi.Error{
			jsonapi.NewError(http.StatusConflict, "INVALID_TYPE", "Invalid resource type", "Expected type 'transition'"),
		})
		return
	}

	cmd := appInstance.TransitionInstanceCommand{
		InstanceID: instanceID,
		Event:      req.Data.Attributes.Event,
		ActorID:    req.Data.Attributes.ActorID,
		Reason:     req.Data.Attributes.Reason,
		Feedback:   req.Data.Attributes.Feedback,
		Metadata:   req.Data.Attributes.Metadata,
		Data:       req.Data.Attributes.Data,
	}

	// Extract roles from JWT context for guard evaluation
	if roles, exists := c.Get("roles"); exists {
		if r, ok := roles.([]string); ok {
			cmd.Roles = r
		}
	}

	result, err := h.transitionUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resource := jsonapi.NewResource("instance", result.ID, map[string]interface{}{
		"workflow_id":    result.WorkflowID,
		"current_state":  result.CurrentState,
		"previous_state": result.PreviousState,
		"status":         result.Status,
		"version":        result.Version,
		"updated_at":     result.UpdatedAt,
	})

	jsonapi.SuccessResponse(c, http.StatusOK, resource, nil, &jsonapi.Links{
		Self: "/api/v1/instances/" + result.ID,
	})
}
