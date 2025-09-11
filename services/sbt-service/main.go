package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/handlers"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/services"
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Get environment variables
	databaseURL := getEnvOrDefault("DATABASE_URL", "mysql://pocw_user:pocw_password@localhost:3306/pocw_db")
	pinataAPIKey := getEnvOrDefault("PINATA_API_KEY", "")
	pinataSecretKey := getEnvOrDefault("PINATA_SECRET_KEY", "")
	baseURL := getEnvOrDefault("BASE_URL", "http://localhost:8080")
	port := getEnvOrDefault("PORT", "8080")
	logLevel := getEnvOrDefault("LOG_LEVEL", "info")

	// Validate required environment variables
	if pinataAPIKey == "" || pinataSecretKey == "" {
		log.Fatal("PINATA_API_KEY and PINATA_SECRET_KEY are required")
	}

	// Set Gin mode
	if logLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	db, err := connectDatabase(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	pinataService := services.NewPinataService(pinataAPIKey, pinataSecretKey)
	pointsServiceURL := getEnvOrDefault("POINTS_SERVICE_URL", "http://localhost:8087")
	metadataService := services.NewMetadataService(db, pinataService, baseURL, pointsServiceURL)

	// Initialize blockchain service and check contract
	blockchainService, err := services.NewBlockchainService()
	if err != nil {
		log.Fatalf("Failed to initialize blockchain service: %v", err)
	}
	defer blockchainService.Close()

	// Check if contract is deployed
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := blockchainService.CheckContractDeployed(ctx); err != nil {
		log.Fatalf("Contract deployment check failed: %v", err)
	}

	// Create routes
	router := setupRouter(metadataService, blockchainService)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("SBT Service starting on port %s", port)
		log.Printf("Base URL: %s", baseURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// connectDatabase connects to database
func connectDatabase(databaseURL string) (*sql.DB, error) {
	// Parse database URL: mysql://user:password@host:port/database
	cfg, err := mysql.ParseDSN(databaseURL[8:]) // Remove "mysql://" prefix
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %v", err)
	}

	// Set connection parameters
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	// Connect to database
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Database connected successfully")
	return db, nil
}

// setupRouter sets up routes
func setupRouter(metadataService *services.MetadataService, blockchainService *services.BlockchainService) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	router.Use(cors.New(config))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"service":   "sbt-service",
			"timestamp": time.Now().Unix(),
		})
	})

	// API route group
	api := router.Group("/api/v1")
	{
		// SBT related routes
		sbt := api.Group("/sbt")
		{
			// Create handlers instance
			sbtHandler := handlers.NewSBTHandler(metadataService, blockchainService)

			// User register SBT
			sbt.POST("/register", sbtHandler.RegisterUser)

			// Get dynamic metadata (for NFT platform calls)
			sbt.GET("/dynamic/:wallet", sbtHandler.GetDynamicMetadata)

			// Get user profile
			sbt.GET("/profile/:wallet", sbtHandler.GetUserProfile)

			// Update user profile
			sbt.PUT("/profile/:wallet", sbtHandler.UpdateUserProfile)

			// Update invite relation
			sbt.PUT("/invite/:wallet", sbtHandler.UpdateInviteRelation)
		}
	}

	return router
}

// getEnvOrDefault gets environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
