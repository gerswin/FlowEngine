package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/http/handler"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/http/middleware"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/security"
)

// Router configures all HTTP routes.
type Router struct {
	workflowHandler *handler.WorkflowHandler
	instanceHandler *handler.InstanceHandler
	tokenService    *security.TokenService
}

// NewRouter creates a new router.
func NewRouter(
	workflowHandler *handler.WorkflowHandler,
	instanceHandler *handler.InstanceHandler,
	tokenService *security.TokenService,
) *Router {
	return &Router{
		workflowHandler: workflowHandler,
		instanceHandler: instanceHandler,
		tokenService:    tokenService,
	}
}

// Setup configures all routes on the gin engine.
func (r *Router) Setup() *gin.Engine {
	router := gin.Default()

	// Add custom middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())

	// Health check (Public)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "FlowEngine",
			"version": "0.1.0",
		})
	})

	// Dev Auth Endpoint (Public) - FOR TESTING ONLY
	router.POST("/api/v1/auth/token", func(c *gin.Context) {
		var req struct {
			UserID string   `json:"user_id"`
			Roles  []string `json:"roles"`
		}
		c.ShouldBindJSON(&req)

		userID := req.UserID
		if userID == "" {
			userID = "admin"
		}
		roles := req.Roles
		if len(roles) == 0 {
			roles = []string{"admin"}
		}

		token, err := r.tokenService.GenerateToken(userID, roles, 24*time.Hour)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to generate token"})
			return
		}
		c.JSON(200, gin.H{"token": token})
	})

	// API v1 routes (Protected)
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(r.tokenService))
	{
		// Workflow routes
		workflows := v1.Group("/workflows")
		{
			workflows.POST("", middleware.RequirePermission(actor.PermCreateWorkflow), r.workflowHandler.CreateWorkflow)
			workflows.POST("/from-yaml", middleware.RequirePermission(actor.PermCreateWorkflow), r.workflowHandler.CreateWorkflowFromYAML)
			workflows.GET("", middleware.RequirePermission(actor.PermReadWorkflow), r.workflowHandler.ListWorkflows)
			workflows.GET("/:id", middleware.RequirePermission(actor.PermReadWorkflow), r.workflowHandler.GetWorkflow)
		}

		// Instance routes
		instances := v1.Group("/instances")
		{
			instances.POST("", middleware.RequirePermission(actor.PermCreateInstance), r.instanceHandler.CreateInstance)
			instances.GET("", middleware.RequirePermission(actor.PermReadInstance), r.instanceHandler.ListInstances)
			instances.GET("/:id", middleware.RequirePermission(actor.PermReadInstance), r.instanceHandler.GetInstance)
			instances.GET("/:id/history", middleware.RequirePermission(actor.PermReadInstance), r.instanceHandler.GetInstanceHistory)
			instances.POST("/:id/transitions", middleware.RequirePermission(actor.PermTransition), r.instanceHandler.TransitionInstance)
			instances.POST("/:id/clone", middleware.RequirePermission(actor.PermCloneInstance), r.instanceHandler.CloneInstance)
		}
	}

	return router
}
