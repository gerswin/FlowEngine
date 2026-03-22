package router

import (
	"time"

	"github.com/gin-gonic/gin"

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
		// Create a dummy token for user "admin" with role "admin"
		token, err := r.tokenService.GenerateToken("admin", []string{"admin"}, 24*time.Hour)
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
			workflows.POST("", r.workflowHandler.CreateWorkflow)
			workflows.POST("/from-yaml", r.workflowHandler.CreateWorkflowFromYAML)
			workflows.GET("", r.workflowHandler.ListWorkflows)
			workflows.GET("/:id", r.workflowHandler.GetWorkflow)
		}

		// Instance routes
		instances := v1.Group("/instances")
		{
			instances.POST("", r.instanceHandler.CreateInstance)
			instances.GET("", r.instanceHandler.ListInstances)
			instances.GET("/:id", r.instanceHandler.GetInstance)
			instances.GET("/:id/history", r.instanceHandler.GetInstanceHistory)
			instances.POST("/:id/transitions", r.instanceHandler.TransitionInstance)
			instances.POST("/:id/clone", r.instanceHandler.CloneInstance)
		}
	}

	return router
}
