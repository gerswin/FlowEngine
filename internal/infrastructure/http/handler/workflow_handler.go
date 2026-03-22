package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	appWorkflow "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/workflow"
	yamlparser "github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/parser/yaml"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

// WorkflowHandler handles HTTP requests for workflows.
type WorkflowHandler struct {
	createUseCase     *appWorkflow.CreateWorkflowUseCase
	createFromYAML    *appWorkflow.CreateWorkflowFromYAMLUseCase
	getUseCase        *appWorkflow.GetWorkflowUseCase
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(
	createUseCase *appWorkflow.CreateWorkflowUseCase,
	createFromYAML *appWorkflow.CreateWorkflowFromYAMLUseCase,
	getUseCase *appWorkflow.GetWorkflowUseCase,
) *WorkflowHandler {
	return &WorkflowHandler{
		createUseCase:     createUseCase,
		createFromYAML:    createFromYAML,
		getUseCase:        getUseCase,
	}
}

// CreateWorkflowRequest represents the request body for creating a workflow.
type CreateWorkflowRequest struct {
	Data struct {
		Type       string `json:"type" binding:"required"`
		Attributes struct {
			Name         string                 `json:"name" binding:"required"`
			Description  string                 `json:"description"`
			InitialState appWorkflow.StateDTO   `json:"initial_state" binding:"required"`
			States       []appWorkflow.StateDTO `json:"states" binding:"required"`
			Events       []appWorkflow.EventDTO `json:"events"`
			CreatedBy    string                 `json:"created_by" binding:"required"`
		} `json:"attributes" binding:"required"`
	} `json:"data" binding:"required"`
}

// CreateWorkflow handles POST /api/v1/workflows
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error()),
		})
		return
	}

	if req.Data.Type != "workflow" {
		jsonapi.ErrorResponse(c, http.StatusConflict, []*jsonapi.Error{
			jsonapi.NewError(http.StatusConflict, "INVALID_TYPE", "Invalid resource type", "Expected type 'workflow'"),
		})
		return
	}

	cmd := appWorkflow.CreateWorkflowCommand{
		Name:         req.Data.Attributes.Name,
		Description:  req.Data.Attributes.Description,
		InitialState: req.Data.Attributes.InitialState,
		States:       req.Data.Attributes.States,
		Events:       req.Data.Attributes.Events,
		CreatedBy:    req.Data.Attributes.CreatedBy,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resource := jsonapi.NewResource("workflow", result.ID, map[string]interface{}{
		"name":        result.Name,
		"description": result.Description,
		"version":     result.Version,
		"created_at":  result.CreatedAt,
		"updated_at":  result.UpdatedAt,
		// Note: States and events could be relationships or attributes depending on complexity.
		// For now, we can keep them as attributes or omit for brevity in list views.
		"states": result.States,
		"events": result.Events,
	})

	jsonapi.SuccessResponse(c, http.StatusCreated, resource, nil, &jsonapi.Links{
		Self: "/api/v1/workflows/" + result.ID,
	})
}

// GetWorkflow handles GET /api/v1/workflows/:id
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	workflowID := c.Param("id")

	result, err := h.getUseCase.Execute(c.Request.Context(), workflowID)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resource := jsonapi.NewResource("workflow", result.ID, map[string]interface{}{
		"name":        result.Name,
		"description": result.Description,
		"version":     result.Version,
		"created_at":  result.CreatedAt,
		"updated_at":  result.UpdatedAt,
		"states":      result.States,
		"events":      result.Events,
	})

	jsonapi.SuccessResponse(c, http.StatusOK, resource, nil, &jsonapi.Links{
		Self: "/api/v1/workflows/" + result.ID,
	})
}

// ListWorkflows handles GET /api/v1/workflows
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	queryOpts := jsonapi.ParseQueryOptions(c)

	result, err := h.getUseCase.ExecuteList(c.Request.Context(), queryOpts.PageNumber, queryOpts.PageSize)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	resources := make([]*jsonapi.Resource, len(result.Items))
	for i, res := range result.Items {
		resources[i] = jsonapi.NewResource("workflow", res.ID, map[string]interface{}{
			"name":        res.Name,
			"description": res.Description,
			"version":     res.Version,
			"created_at":  res.CreatedAt,
			"updated_at":  res.UpdatedAt,
		})
		resources[i].Links = &jsonapi.Links{
			Self: "/api/v1/workflows/" + res.ID,
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

	jsonapi.SuccessResponse(c, http.StatusOK, resources, meta, &jsonapi.Links{
		Self: c.Request.URL.String(),
	})
}

// CreateWorkflowFromYAMLRequest represents the request for YAML workflow creation.
type CreateWorkflowFromYAMLRequest struct {
	Data struct {
		Type       string `json:"type" binding:"required"`
		Attributes struct {
			YAML      string `json:"yaml" binding:"required"`
			CreatedBy string `json:"created_by" binding:"required"`
		} `json:"attributes" binding:"required"`
	} `json:"data" binding:"required"`
}

// CreateWorkflowFromYAML handles POST /api/v1/workflows/from-yaml
// Accepts YAML content either as JSON body or multipart form file upload.
func (h *WorkflowHandler) CreateWorkflowFromYAML(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")

	var yamlContent []byte
	var createdBy string

	// Handle multipart form (file upload)
	if strings.HasPrefix(contentType, "multipart/form-data") {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
				jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Failed to read file", err.Error()),
			})
			return
		}
		defer file.Close()

		yamlContent, err = io.ReadAll(file)
		if err != nil {
			jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
				jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Failed to read file content", err.Error()),
			})
			return
		}

		createdBy = c.PostForm("created_by")
		if createdBy == "" {
			// Try to get from auth context
			if userID, exists := c.Get("userID"); exists {
				createdBy = userID.(string)
			}
		}
	} else {
		// Handle JSON body with embedded YAML
		var req CreateWorkflowFromYAMLRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
				jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error()),
			})
			return
		}

		if req.Data.Type != "workflow-yaml" {
			jsonapi.ErrorResponse(c, http.StatusConflict, []*jsonapi.Error{
				jsonapi.NewError(http.StatusConflict, "INVALID_TYPE", "Invalid resource type", "Expected type 'workflow-yaml'"),
			})
			return
		}

		yamlContent = []byte(req.Data.Attributes.YAML)
		createdBy = req.Data.Attributes.CreatedBy
	}

	// Validate inputs
	if len(yamlContent) == 0 {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "YAML content is required", ""),
		})
		return
	}

	if createdBy == "" {
		jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
			jsonapi.NewError(http.StatusBadRequest, "INVALID_REQUEST", "created_by is required", ""),
		})
		return
	}

	// Execute use case
	cmd := appWorkflow.CreateWorkflowFromYAMLCommand{
		YAMLContent: yamlContent,
		CreatedBy:   createdBy,
	}

	result, err := h.createFromYAML.Execute(c.Request.Context(), cmd)
	if err != nil {
		// Handle specific error types
		switch e := err.(type) {
		case *yamlparser.ValidationErrors:
			errors := make([]*jsonapi.Error, 0, len(e.Errors))
			for _, ve := range e.Errors {
				errors = append(errors, jsonapi.NewError(
					http.StatusUnprocessableEntity,
					"VALIDATION_ERROR",
					"Workflow validation failed",
					ve.Error(),
				))
			}
			jsonapi.ErrorResponse(c, http.StatusUnprocessableEntity, errors)
			return
		case yamlparser.InvalidYAMLError:
			jsonapi.ErrorResponse(c, http.StatusBadRequest, []*jsonapi.Error{
				jsonapi.NewError(http.StatusBadRequest, "YAML_SYNTAX_ERROR", "Invalid YAML syntax", e.Error()),
			})
			return
		default:
			handleDomainError(c, err)
			return
		}
	}

	// Build response
	attributes := map[string]interface{}{
		"name":          result.Name,
		"description":   result.Description,
		"version":       result.Version,
		"initial_state": result.InitialState,
		"states_count":  result.StatesCount,
		"events_count":  result.EventsCount,
		"created_at":    result.CreatedAt,
	}

	if len(result.Warnings) > 0 {
		attributes["warnings"] = result.Warnings
	}

	resource := jsonapi.NewResource("workflow", result.ID, attributes)

	meta := map[string]interface{}{
		"source": "yaml",
	}

	jsonapi.SuccessResponse(c, http.StatusCreated, resource, meta, &jsonapi.Links{
		Self: "/api/v1/workflows/" + result.ID,
	})
}
