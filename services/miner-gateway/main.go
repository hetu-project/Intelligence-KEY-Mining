package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/protocol"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/config"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/handlers"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/middleware"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/verifiers"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize database
	db, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 3. Initialize validator registry
	verifierRegistry := verifiers.NewVerifierRegistry()
	verifierRegistry.RegisterVerifier("twitter_retweet", verifiers.NewTwitterVerifier(cfg.TwitterMiddleLayerURL, cfg.TwitterAPIKey))
	verifierRegistry.RegisterVerifier("task_creation", verifiers.NewTaskCreationVerifier())
	verifierRegistry.RegisterVerifier("batch_verification", verifiers.NewBatchVerificationVerifier(cfg.TwitterMiddleLayerURL, cfg.TwitterAPIKey))

	// 4. Initialize services
	vlcService := services.NewVLCService()
	vlcStrategy := services.NewDefaultVLCStrategy()
	enhancedVLCService := services.NewEnhancedVLCService(vlcStrategy)

	// Create network configuration, convert ValidatorEndpoint types
	var protocolEndpoints []protocol.ValidatorEndpoint
	for _, ep := range cfg.ValidatorEndpoints {
		protocolEndpoints = append(protocolEndpoints, protocol.ValidatorEndpoint{
			ID:       ep.ID,
			Role:     ep.Role,
			URL:      ep.URL,
			Weight:   ep.Weight,
			Priority: ep.Priority,
		})
	}

	networkConfig := &protocol.NetworkConfig{
		ValidatorEndpoints: protocolEndpoints,
		RequestTimeout:     30 * time.Second,
		MaxRetries:         3,
		RetryInterval:      5 * time.Second,
	}

	validatorClient := services.NewValidatorClient(networkConfig, "miner-1")
	taskService := services.NewTaskService(db, verifierRegistry, vlcService, validatorClient, cfg.MinerPrivateKey, "miner-1")

	// 5. Initialize async services
	pointsServiceURL := os.Getenv("POINTS_SERVICE_URL")
	if pointsServiceURL == "" {
		pointsServiceURL = "http://localhost:8087" // Default points service URL
	}
	batchVerifier := services.NewBatchVerifier(taskService, enhancedVLCService, pointsServiceURL, 5) // 5 workers
	taskCreationVerifier := verifiers.NewTaskCreationVerifier()
	validatorScheduler := services.NewValidatorScheduler(taskService, taskCreationVerifier, batchVerifier)

	// Start async services
	ctx := context.Background()
	if err := batchVerifier.Start(ctx); err != nil {
		log.Fatalf("Failed to start batch verifier: %v", err)
	}
	if err := validatorScheduler.Start(ctx); err != nil {
		log.Fatalf("Failed to start validator scheduler: %v", err)
	}

	// 6. Initialize handlers
	taskHandler := handlers.NewTaskHandler(taskService)
	taskCreationHandler := handlers.NewTaskCreationHandler(taskService)
	batchVerificationHandler := handlers.NewBatchVerificationHandler(taskService, batchVerifier)
	healthHandler := handlers.NewHealthHandler(db)

	// 5. Setup routes
	router := setupRoutes(taskHandler, taskCreationHandler, batchVerificationHandler, healthHandler)

	// 6. Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("MinerGateway server started on port %s", cfg.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	// Stop async services
	batchVerifier.Stop()
	validatorScheduler.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}

func setupRoutes(
	taskHandler *handlers.TaskHandler,
	taskCreationHandler *handlers.TaskCreationHandler,
	batchVerificationHandler *handlers.BatchVerificationHandler,
	healthHandler *handlers.HealthHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit())

	// Health check
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Legacy task related
		tasks := v1.Group("/tasks")
		{
			tasks.POST("/submit", taskHandler.SubmitTask)
			tasks.GET("/status/:id", taskHandler.GetTaskStatus)
			tasks.GET("/user/:wallet", taskHandler.GetUserTasks)
		}

		// Task creation related
		taskCreation := v1.Group("/task-creation")
		{
			taskCreation.POST("/create", taskCreationHandler.CreateTwitterTask)
			taskCreation.GET("/status/:id", taskCreationHandler.GetTaskCreationStatus)
			taskCreation.GET("/user/:wallet", taskCreationHandler.ListUserTaskCreations)
			taskCreation.GET("/stats", taskCreationHandler.GetTaskCreationStats)
		}

		// Batch verification related
		batchVerification := v1.Group("/batch-verification")
		{
			batchVerification.POST("/verify", batchVerificationHandler.BatchVerifyTasks)
			batchVerification.GET("/status/:id", batchVerificationHandler.GetBatchVerificationStatus)
			batchVerification.GET("/user/:wallet", batchVerificationHandler.ListBatchVerifications)
			batchVerification.GET("/stats", batchVerificationHandler.GetBatchVerificationStats)
		}
	}

	return router
}
