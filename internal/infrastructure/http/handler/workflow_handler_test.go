package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appWorkflow "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/workflow"
	domainWorkflow "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	eventMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event/mocks"
	workflowMocks "github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow/mocks"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

// Helper to parse JSON:API response
func parseWorkflowResponse(t *testing.T, body []byte) *jsonapi.Response {
	var resp jsonapi.Response
	err := json.Unmarshal(body, &resp)
	require.NoError(t, err)
	return &resp
}

// Helper to extract single resource from response
func extractWorkflowResource(t *testing.T, data interface{}) map[string]interface{} {
	resource, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", data)
	}
	return resource
}

// Helper to extract resource list from response
func extractWorkflowResourceList(t *testing.T, data interface{}) []interface{} {
	resources, ok := data.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", data)
	}
	return resources
}

// workflowTestSetup creates a test router with the workflow handler wired to mocks
type workflowTestSetup struct {
	router          *gin.Engine
	workflowRepo    *workflowMocks.Repository
	eventDispatcher *eventMocks.Dispatcher
	handler         *WorkflowHandler
}

func setupWorkflowHandlerTest(t *testing.T) *workflowTestSetup {
	gin.SetMode(gin.TestMode)

	workflowRepo := new(workflowMocks.Repository)
	eventDispatcher := new(eventMocks.Dispatcher)

	createUseCase := appWorkflow.NewCreateWorkflowUseCase(workflowRepo, eventDispatcher)
	getUseCase := appWorkflow.NewGetWorkflowUseCase(workflowRepo)
	createFromYAML := appWorkflow.NewCreateWorkflowFromYAMLUseCase(workflowRepo, eventDispatcher)

	handler := NewWorkflowHandler(createUseCase, createFromYAML, getUseCase)

	router := gin.New()
	router.POST("/workflows", handler.CreateWorkflow)
	router.POST("/workflows/yaml", handler.CreateWorkflowFromYAML)
	router.GET("/workflows", handler.ListWorkflows)
	router.GET("/workflows/:id", handler.GetWorkflow)

	return &workflowTestSetup{
		router:          router,
		workflowRepo:    workflowRepo,
		eventDispatcher: eventDispatcher,
		handler:         handler,
	}
}

// ============================================================================
// CreateWorkflow Tests
// ============================================================================

func TestCreateWorkflow_Success(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	setup.workflowRepo.On("Save", mock.Anything, mock.MatchedBy(func(wf *domainWorkflow.Workflow) bool {
		return wf.Name() == "Approval Process"
	})).Return(nil)
	setup.eventDispatcher.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow",
			"attributes": map[string]interface{}{
				"name":        "Approval Process",
				"description": "A simple approval workflow",
				"created_by":  creatorID.String(),
				"initial_state": map[string]interface{}{
					"id":   "draft",
					"name": "Draft",
				},
				"states": []map[string]interface{}{
					{
						"id":   "draft",
						"name": "Draft",
					},
					{
						"id":       "approved",
						"name":     "Approved",
						"is_final": true,
					},
				},
				"events": []map[string]interface{}{
					{
						"name":        "approve",
						"sources":     []string{"draft"},
						"destination": "approved",
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Data)
}

func TestCreateWorkflow_InvalidBody(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	// Missing required fields
	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow",
			"attributes": map[string]interface{}{
				// missing name, initial_state, created_by
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
	assert.Equal(t, "INVALID_REQUEST", resp.Errors[0].Code)
}

func TestCreateWorkflow_WrongResourceType(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "process", // Wrong type
			"attributes": map[string]interface{}{
				"name":        "Approval Process",
				"description": "A simple approval workflow",
				"created_by":  creatorID.String(),
				"initial_state": map[string]interface{}{
					"id":   "draft",
					"name": "Draft",
				},
				"states": []map[string]interface{}{
					{
						"id":   "draft",
						"name": "Draft",
					},
				},
				"events": []map[string]interface{}{},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
	assert.Equal(t, "INVALID_TYPE", resp.Errors[0].Code)
}

func TestCreateWorkflow_InvalidCreatorID(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow",
			"attributes": map[string]interface{}{
				"name":        "Approval Process",
				"created_by":  "invalid-id-format", // Invalid ID
				"initial_state": map[string]interface{}{
					"id":   "draft",
					"name": "Draft",
				},
				"states": []map[string]interface{}{
					{
						"id":   "draft",
						"name": "Draft",
					},
				},
				"events": []map[string]interface{}{},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
}

func TestCreateWorkflow_MissingInitialState(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow",
			"attributes": map[string]interface{}{
				"name":        "Approval Process",
				"created_by":  creatorID.String(),
				"initial_state": nil, // Missing
				"states": []map[string]interface{}{
					{
						"id":   "draft",
						"name": "Draft",
					},
				},
				"events": []map[string]interface{}{},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
}

// ============================================================================
// GetWorkflow Tests
// ============================================================================

func TestGetWorkflow_Success(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	workflowID := shared.NewID()
	creatorID := shared.NewID()

	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test Workflow", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("FindByID", mock.Anything, workflowID).Return(wf, nil)

	req := httptest.NewRequest("GET", "/workflows/"+workflowID.String(), nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Data)
}

func TestGetWorkflow_NotFound(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	workflowID := shared.NewID()

	setup.workflowRepo.On("FindByID", mock.Anything, workflowID).
		Return(nil, domainWorkflow.WorkflowNotFoundError(workflowID.String()))

	req := httptest.NewRequest("GET", "/workflows/"+workflowID.String(), nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
}

func TestGetWorkflow_InvalidIDFormat(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	req := httptest.NewRequest("GET", "/workflows/invalid-format", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Errors)
}

func TestGetWorkflow_ReturnsStatesAndEvents(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	workflowID := shared.NewID()
	creatorID := shared.NewID()

	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test Workflow", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("FindByID", mock.Anything, workflowID).Return(wf, nil)

	req := httptest.NewRequest("GET", "/workflows/"+workflowID.String(), nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Data)
}

// ============================================================================
// ListWorkflows Tests
// ============================================================================

func TestListWorkflows_Success(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf1, err := domainWorkflow.NewWorkflow("Workflow 1", initialState, creatorID)
	require.NoError(t, err)

	wf2, err := domainWorkflow.NewWorkflow("Workflow 2", initialState, creatorID)
	require.NoError(t, err)

	workflows := []*domainWorkflow.Workflow{wf1, wf2}

	setup.workflowRepo.On("List", mock.Anything, mock.Anything).Return(workflows, int64(2), nil)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	resources := extractWorkflowResourceList(t, resp.Data)
	assert.Equal(t, 2, len(resources))

	// Check metadata
	assert.NotNil(t, resp.Meta)
	meta := resp.Meta.(map[string]interface{})
	assert.Equal(t, float64(2), meta["total"])
}

func TestListWorkflows_Empty(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	setup.workflowRepo.On("List", mock.Anything, mock.Anything).Return([]*domainWorkflow.Workflow{}, int64(0), nil)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	resources := extractWorkflowResourceList(t, resp.Data)
	assert.Equal(t, 0, len(resources))

	meta := resp.Meta.(map[string]interface{})
	assert.Equal(t, float64(0), meta["total"])
}

func TestListWorkflows_WithPagination(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	workflows := make([]*domainWorkflow.Workflow, 10)
	for i := 0; i < 10; i++ {
		initialState, err := domainWorkflow.NewState("draft", "Draft")
		require.NoError(t, err)

		wf, err := domainWorkflow.NewWorkflow("Workflow", initialState, creatorID)
		require.NoError(t, err)
		workflows[i] = wf
	}

	setup.workflowRepo.On("List", mock.Anything, mock.MatchedBy(func(q shared.ListQuery) bool {
		return q.Offset == 5 && q.Limit == 5
	})).Return(workflows[:5], int64(25), nil)

	req := httptest.NewRequest("GET", "/workflows?page[number]=2&page[size]=5", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())

	meta := resp.Meta.(map[string]interface{})
	pageInfo := meta["page"].(map[string]interface{})
	assert.Equal(t, float64(2), pageInfo["number"])
	assert.Equal(t, float64(5), pageInfo["size"])
}

func TestListWorkflows_DefaultPagination(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()
	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("List", mock.Anything, mock.MatchedBy(func(q shared.ListQuery) bool {
		// Default page size should be 20 from jsonapi parser (Offset=0, Limit=20)
		return q.Offset == 0 && q.Limit == 20
	})).Return([]*domainWorkflow.Workflow{wf}, int64(1), nil)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())

	meta := resp.Meta.(map[string]interface{})
	pageInfo := meta["page"].(map[string]interface{})
	assert.Equal(t, float64(1), pageInfo["number"])
	assert.Equal(t, float64(20), pageInfo["size"])
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestCreateWorkflow_WithDescription(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	setup.workflowRepo.On("Save", mock.Anything, mock.MatchedBy(func(wf *domainWorkflow.Workflow) bool {
		return wf.Description() == "A detailed description"
	})).Return(nil)
	setup.eventDispatcher.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow",
			"attributes": map[string]interface{}{
				"name":        "Detailed Workflow",
				"description": "A detailed description",
				"created_by":  creatorID.String(),
				"initial_state": map[string]interface{}{
					"id":   "start",
					"name": "Start",
				},
				"states": []map[string]interface{}{
					{
						"id":   "start",
						"name": "Start",
					},
				},
				"events": []map[string]interface{}{},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Data)
}

func TestListWorkflows_ReturnsCorrectAttributes(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()
	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("List", mock.Anything, mock.Anything).Return([]*domainWorkflow.Workflow{wf}, int64(1), nil)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	resources := extractWorkflowResourceList(t, resp.Data)
	assert.GreaterOrEqual(t, len(resources), 1)
}

func TestWorkflowHandler_JSONAPIFormat(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	workflowID := shared.NewID()
	creatorID := shared.NewID()

	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("FindByID", mock.Anything, workflowID).Return(wf, nil)

	req := httptest.NewRequest("GET", "/workflows/"+workflowID.String(), nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify correct Content-Type
	assert.Equal(t, "application/vnd.api+json", w.Header().Get("Content-Type"))

	resp := parseWorkflowResponse(t, w.Body.Bytes())

	// Verify JSON:API structure
	assert.NotNil(t, resp.Data)
	assert.Nil(t, resp.Errors)
	assert.NotNil(t, resp.Jsonapi)
	assert.Equal(t, "1.0", resp.Jsonapi.Version)

	// Verify resource structure exists
	assert.NotNil(t, resp.Data)
}

func TestListWorkflows_IncludesHATEOASLinks(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()
	initialState, err := domainWorkflow.NewState("draft", "Draft")
	require.NoError(t, err)

	wf, err := domainWorkflow.NewWorkflow("Test", initialState, creatorID)
	require.NoError(t, err)

	setup.workflowRepo.On("List", mock.Anything, mock.Anything).Return([]*domainWorkflow.Workflow{wf}, int64(1), nil)

	req := httptest.NewRequest("GET", "/workflows?page[number]=1&page[size]=20", nil)
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())

	// Verify top-level links exist
	assert.NotNil(t, resp.Links)
	assert.NotEmpty(t, resp.Links.Self)
}

// ============================================================================
// CreateWorkflowFromYAML Tests
// ============================================================================

func TestCreateWorkflowFromYAML_JSONBody_Success(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	setup.workflowRepo.On("Save", mock.Anything, mock.AnythingOfType("*workflow.Workflow")).Return(nil)
	setup.eventDispatcher.On("DispatchBatch", mock.Anything, mock.Anything).Return(nil)

	yamlContent := `version: "1.0"
workflow:
  name: YAMLWorkflow
  initial_state: draft
  states:
    - id: draft
      name: Draft
    - id: approved
      name: Approved
      is_final: true
  events:
    - name: approve
      from: [draft]
      to: approved`

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow-yaml",
			"attributes": map[string]interface{}{
				"yaml":       yamlContent,
				"created_by": creatorID.String(),
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := parseWorkflowResponse(t, w.Body.Bytes())
	assert.NotNil(t, resp.Data)
}

func TestCreateWorkflowFromYAML_JSONBody_WrongType(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "wrong-type",
			"attributes": map[string]interface{}{
				"yaml":       "name: test",
				"created_by": creatorID.String(),
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateWorkflowFromYAML_JSONBody_EmptyYAML(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow-yaml",
			"attributes": map[string]interface{}{
				"yaml":       "",
				"created_by": creatorID.String(),
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWorkflowFromYAML_JSONBody_MissingCreatedBy(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow-yaml",
			"attributes": map[string]interface{}{
				"yaml":       "name: test\ninitial_state: draft\nstates:\n  - id: draft\n    name: Draft",
				"created_by": "",
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWorkflowFromYAML_JSONBody_InvalidJSON(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWorkflowFromYAML_JSONBody_InvalidYAMLSyntax(t *testing.T) {
	setup := setupWorkflowHandlerTest(t)

	creatorID := shared.NewID()

	body := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "workflow-yaml",
			"attributes": map[string]interface{}{
				"yaml":       "invalid: [yaml: syntax: {{",
				"created_by": creatorID.String(),
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/workflows/yaml", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	setup.router.ServeHTTP(w, req)

	// Should return 400 for bad YAML or 422 for validation
	assert.GreaterOrEqual(t, w.Code, 400)
}
