package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appInstance "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/instance"
	appWorkflow "github.com/LaFabric-LinkTIC/FlowEngine/internal/application/workflow"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/instance"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/timer"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/workflow"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/cache"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/http/handler"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/http/router"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/messaging"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/persistence/memory"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/persistence/postgres"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/scheduler"
	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/security"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/logger"
)

func main() {
	// Initialize Logger
	logger.Init()
	logger.Info("🚀 Starting FlowEngine API Server...")

	// --- Security ---

tokenService := security.NewTokenService()

	// --- Context for initial setup and graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Persistence & Cache Initialization ---
	var workflowRepo workflow.Repository
	var instanceRepo instance.Repository
	var timerRepo timer.Repository // Add Timer Repo
	var cch cache.Cache

	// Try to initialize Redis cache
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		logger.Info("⚡ Connecting to Redis cache...", "addr", redisAddr)
		redisConfig := cache.GetRedisConfigFromEnv()
		redisClient, err := cache.GetRedisClient(ctx, redisConfig)
		if err != nil {
			logger.Warn("❌ Failed to connect to Redis. Proceeding without cache.", "error", err)
			cch = nil
		} else {
			cch = cache.NewRedisCache(redisClient)
			logger.Info("✅ Connected to Redis cache")
			defer cache.CloseRedisClient()
		}
	} else {
		logger.Info("ℹ️ REDIS_ADDR not set. Running without Redis cache.")
		cch = nil
	}

	// Try to initialize PostgreSQL persistence
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost != "" {
		logger.Info("🔌 Connecting to PostgreSQL...", "host", dbHost)
		dbConfig := &postgres.DBConfig{
			Host:     dbHost,
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:   getEnv("POSTGRES_DB", "flowengine"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		}

		dbPool, err := postgres.GetDBPool(ctx, dbConfig)
		if err != nil {
			logger.Error("❌ Failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer postgres.CloseDBPool()
		logger.Info("✅ Connected to PostgreSQL")

		workflowRepo = postgres.NewWorkflowRepository(dbPool, cch)
		instanceRepo = postgres.NewInstanceRepository(dbPool, cch)
		timerRepo = postgres.NewTimerRepository(dbPool) // Initialize Timer Repo
	} else {
		logger.Warn("⚠️ POSTGRES_HOST not set. Using In-Memory Persistence (Data will be lost on restart)")
		workflowRepo = memory.NewWorkflowInMemoryRepository()
		instanceRepo = memory.NewInstanceInMemoryRepository()
		timerRepo = memory.NewTimerInMemoryRepository()
		cch = nil
	}

	// --- Event System ---
	inMemoryDispatcher := event.NewInMemoryDispatcher()
	logDispatcher := messaging.NewLogDispatcher()
	eventBus := messaging.NewMultiDispatcher(inMemoryDispatcher, logDispatcher)

	// --- Use Cases ---

	createWorkflowUseCase := appWorkflow.NewCreateWorkflowUseCase(workflowRepo, eventBus)
	createWorkflowFromYAMLUseCase := appWorkflow.NewCreateWorkflowFromYAMLUseCase(workflowRepo, eventBus)
	getWorkflowUseCase := appWorkflow.NewGetWorkflowUseCase(workflowRepo)

	createInstanceUseCase := appInstance.NewCreateInstanceUseCase(workflowRepo, instanceRepo, eventBus)
	getInstanceUseCase := appInstance.NewGetInstanceUseCase(instanceRepo)
	transitionEngine := instance.NewEngine()
	transitionInstanceUseCase := appInstance.NewTransitionInstanceUseCase(workflowRepo, instanceRepo, eventBus, transitionEngine)
	cloneInstanceUseCase := appInstance.NewCloneInstanceUseCase(instanceRepo, workflowRepo, eventBus)

	// --- Scheduler ---
	if timerRepo != nil {
		sched := scheduler.NewWorker(timerRepo, transitionInstanceUseCase, 10*time.Second)
		sched.Start()
		defer sched.Stop()
	} else {
		logger.Warn("⚠️ Scheduler disabled (No Timer Persistence)")
	}

	// --- HTTP Layer ---
	// Handlers
	workflowHandler := handler.NewWorkflowHandler(createWorkflowUseCase, createWorkflowFromYAMLUseCase, getWorkflowUseCase)
	instanceHandler := handler.NewInstanceHandler(createInstanceUseCase, getInstanceUseCase, transitionInstanceUseCase, cloneInstanceUseCase)

	// Router
	routerConfig := router.NewRouter(workflowHandler, instanceHandler, tokenService)
	ginRouter := routerConfig.Setup()

	// Server Config
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        ginRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// --- Start Server ---
	go func() {
		logger.Info("✅ FlowEngine API Server listening", "port", port)
		if dbHost != "" {
			logger.Info("💾 Persistence: PostgreSQL")
		} else {
			logger.Info("💾 Persistence: In-Memory")
		}
		if cch != nil {
			logger.Info("🚀 Cache: Redis")
		} else {
			logger.Info("🐌 Cache: Disabled")
		}
		
		fmt.Println("📋 Available endpoints:")
		fmt.Println("   GET  /health")
		fmt.Println("   POST /api/v1/workflows")
		fmt.Println("   POST /api/v1/workflows/from-yaml")
		fmt.Println("   GET  /api/v1/workflows")
		fmt.Println("   GET  /api/v1/workflows/:id")
		fmt.Println("   POST /api/v1/instances")
		fmt.Println("   GET  /api/v1/instances")
		fmt.Println("   GET  /api/v1/instances/:id")
		fmt.Println("   POST /api/v1/instances/:id/transitions")
		fmt.Println("   POST /api/v1/instances/:id/clone")
		fmt.Println()
		fmt.Println("Press Ctrl+C to stop")
		fmt.Println("========================================")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("\n🛑 Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("✅ Server exited gracefully")
}

// getEnv gets an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}