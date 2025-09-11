package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strconv"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/crypto"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
)

// Config holds all configuration for the validator service
type Config struct {
	Port                string               `json:"port"`
	ValidatorID         string               `json:"validator_id"`
	ValidatorRole       models.ValidatorRole `json:"validator_role"`
	ValidatorWeight     float64              `json:"validator_weight"`
	ValidatorPrivateKey *ecdsa.PrivateKey    `json:"-"`
	DatabaseURL         string               `json:"database_url"`
	DgraphURL           string               `json:"dgraph_url"`
	LogLevel            string               `json:"log_level"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		ValidatorID: getEnv("VALIDATOR_ID", ""),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DgraphURL:   getEnv("DGRAPH_URL", "localhost:9080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	// Load validator role
	roleStr := getEnv("VALIDATOR_ROLE", "")
	switch roleStr {
	case "ui_validator":
		config.ValidatorRole = models.UIValidatorRole
	case "format_validator":
		config.ValidatorRole = models.FormatValidatorRole
	case "semantic_validator":
		config.ValidatorRole = models.SemanticValidatorRole
	default:
		return nil, fmt.Errorf("invalid validator role: %s", roleStr)
	}

	// Load weight
	weightStr := getEnv("VALIDATOR_WEIGHT", "")
	if weightStr == "" {
		return nil, fmt.Errorf("VALIDATOR_WEIGHT is required")
	}

	weight, err := strconv.ParseFloat(weightStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validator weight: %v", err)
	}
	config.ValidatorWeight = weight

	// Load private key
	privateKeyHex := getEnv("VALIDATOR_PRIVATE_KEY", "")
	if privateKeyHex == "" {
		return nil, fmt.Errorf("VALIDATOR_PRIVATE_KEY is required")
	}

	privateKey, err := crypto.LoadPrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}
	config.ValidatorPrivateKey = privateKey

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ValidatorID == "" {
		return fmt.Errorf("VALIDATOR_ID is required")
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.ValidatorWeight <= 0 || c.ValidatorWeight > 1 {
		return fmt.Errorf("validator weight must be between 0 and 1, got: %.2f", c.ValidatorWeight)
	}

	return nil
}

// GetValidatorConfig returns the validator configuration
func (c *Config) GetValidatorConfig() *models.ValidatorConfig {
	return &models.ValidatorConfig{
		ID:         c.ValidatorID,
		Role:       c.ValidatorRole,
		Weight:     c.ValidatorWeight,
		PrivateKey: c.ValidatorPrivateKey,
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
