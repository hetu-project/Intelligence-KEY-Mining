package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/handlers"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/services"
)

func main() {
	// Initialize database connection
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize points configuration
	config := models.DefaultPointsConfig()

	// Override configuration from environment variables (if exists)
	if poolStr := os.Getenv("POINTS_TOTAL_POOL"); poolStr != "" {
		// Can add environment variable parsing logic
	}

	// Initialize services
	pointsService := services.NewPointsService(db, config)

	// Initialize handlers
	pointsHandler := handlers.NewPointsHandler(pointsService)

	// Initialize Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "points-service",
		})
	})

	// Register API routes
	api := router.Group("/api/v1")
	pointsHandler.RegisterRoutes(api)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Points Service starting on port %s", port)
	log.Printf("Points configuration: %+v", config)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDatabase initializes database connection
func initDatabase() (*sql.DB, error) {
	// Get database configuration from environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "hetu_key_mining")

	dsn := dbUser + ":" + dbPass + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Set connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Printf("Database connected successfully to %s:%s/%s", dbHost, dbPort, dbName)
	return db, nil
}

// corsMiddleware CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// getEnv gets environment variable, returns default value if not exists
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
