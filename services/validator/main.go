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
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/config"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/handlers"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/middleware"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/plugins"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/services"
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

	// 3. Initialize services
	vlcService := services.NewVLCService(cfg.ValidatorID)

	// Create validator configuration
	validatorConfig := &models.ValidatorConfig{
		ID:         cfg.ValidatorID,
		Role:       cfg.ValidatorRole,
		Weight:     cfg.ValidatorWeight,
		PrivateKey: cfg.ValidatorPrivateKey,
	}

	// Create plugins
	qualityAssessor := plugins.NewTwitterQualityAssessor(cfg.ValidatorRole)
	formatValidator := plugins.NewTwitterFormatValidator()

	validationService := services.NewValidationService(validatorConfig, vlcService, qualityAssessor, formatValidator)

	// 4. Initialize handlers
	validationHandler := handlers.NewValidationHandler(validationService)
	healthHandler := handlers.NewHealthHandler(db)

	// 5. Setup routes
	router := setupRoutes(validationHandler, healthHandler)

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

	log.Printf("Validator server (%s) started on port %s", cfg.ValidatorID, cfg.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}

func setupRoutes(validationHandler *handlers.ValidationHandler, healthHandler *handlers.HealthHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Validation related
		v1.POST("/validate", validationHandler.ValidateTask)
		v1.GET("/config", validationHandler.GetConfig)
	}

	return router
}
