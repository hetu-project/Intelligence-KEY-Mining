package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/crypto"
)

// Config holds all configuration for the miner gateway service
type Config struct {
	Port                         string              `json:"port"`
	DatabaseURL                  string              `json:"database_url"`
	RedisURL                     string              `json:"redis_url"`
	DgraphURL                    string              `json:"dgraph_url"`
	TwitterMiddleLayerURL        string              `json:"twitter_middle_layer_url"`
	TwitterAPIKey                string              `json:"twitter_api_key"`
	TwitterRetweetCheckURL       string              `json:"twitter_retweet_check_url"`
	MinerPrivateKey              *ecdsa.PrivateKey   `json:"-"`
	ValidatorEndpoints           []ValidatorEndpoint `json:"validator_endpoints"`
	LogLevel                     string              `json:"log_level"`
	ValidatorPollIntervalSeconds int                 `json:"validator_poll_interval_seconds"`
}

// ValidatorEndpoint represents a validator service endpoint
type ValidatorEndpoint struct {
	ID       string  `json:"id"`
	Role     string  `json:"role"`
	URL      string  `json:"url"`
	Weight   float64 `json:"weight"`
	Priority int     `json:"priority"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Parse poll interval from environment variable
	pollIntervalStr := getEnv("VALIDATOR_POLL_INTERVAL_SECONDS", "7200") // 2 hours default
	pollInterval, err := strconv.Atoi(pollIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid VALIDATOR_POLL_INTERVAL_SECONDS: %v", err)
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("VALIDATOR_POLL_INTERVAL_SECONDS must be positive, got: %d", pollInterval)
	}

	config := &Config{
		Port:                         getEnv("PORT", "8080"),
		DatabaseURL:                  getEnv("DATABASE_URL", ""),
		RedisURL:                     getEnv("REDIS_URL", "redis://localhost:6379"),
		DgraphURL:                    getEnv("DGRAPH_URL", "localhost:9080"),
		TwitterMiddleLayerURL:        getEnv("TWITTER_MIDDLE_LAYER_URL", ""),
		TwitterAPIKey:                getEnv("TWITTER_API_KEY", ""),
		TwitterRetweetCheckURL:       getEnv("TWITTER_RETWEET_CHECK_URL", ""),
		LogLevel:                     getEnv("LOG_LEVEL", "info"),
		ValidatorPollIntervalSeconds: pollInterval,
	}

	// Load private key
	privateKeyHex := getEnv("MINER_PRIVATE_KEY", "")
	if privateKeyHex == "" {
		return nil, fmt.Errorf("MINER_PRIVATE_KEY is required")
	}

	privateKey, err := crypto.LoadPrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}
	config.MinerPrivateKey = privateKey

	// Load validator endpoints
	endpointsJSON := getEnv("VALIDATOR_ENDPOINTS", "")
	if endpointsJSON == "" {
		return nil, fmt.Errorf("VALIDATOR_ENDPOINTS is required")
	}

	if err := json.Unmarshal([]byte(endpointsJSON), &config.ValidatorEndpoints); err != nil {
		return nil, fmt.Errorf("failed to parse validator endpoints: %v", err)
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	// TwitterMiddleLayerURL and TwitterAPIKey are no longer required
	// Twitter verification is now handled by TwitterVerificationService with TwitterRetweetCheckURL

	if len(c.ValidatorEndpoints) == 0 {
		return fmt.Errorf("at least one validator endpoint is required")
	}

	// Validate total weight
	totalWeight := 0.0
	for _, endpoint := range c.ValidatorEndpoints {
		totalWeight += endpoint.Weight
	}

	if totalWeight < 0.99 || totalWeight > 1.01 { // Allow floating point error
		return fmt.Errorf("validator weights must sum to 1.0, got: %.2f", totalWeight)
	}

	return nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
