package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the router worker
type Config struct {
	// Worker configuration
	WorkerID string `env:"WORKER_ID" envDefault:"router-1"`

	// Redis configuration
	RedisAddr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisPassword string `env:"REDIS_PASS" envDefault:""`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	// Stream configuration
	StreamKey      string `env:"STREAM_KEY" envDefault:"router.work"`
	ConsumerGroup  string `env:"CONSUMER_GROUP" envDefault:"router-workers"`
	ResultStream   string `env:"RESULT_STREAM" envDefault:"router.decided"`
	BlockTime      time.Duration `env:"BLOCK_TIME" envDefault:"1s"`
	MaxRetries     int    `env:"MAX_RETRIES" envDefault:"3"`

	// LLM configuration
	LLMProvider string `env:"LLM_PROVIDER" envDefault:"anthropic"`
	LLMAPIKey   string `env:"LLM_API_KEY"`
	LLMModel    string `env:"LLM_MODEL" envDefault:"claude-sonnet-4-20250514"`
	LLMTimeout  time.Duration `env:"LLM_TIMEOUT" envDefault:"30s"`

	// CEL configuration
	CELEnabled bool `env:"CEL_ENABLED" envDefault:"true"`

	// Health check configuration
	HealthPort int `env:"HEALTH_PORT" envDefault:"8082"`

	// Logging configuration
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WorkerID == "" {
		return fmt.Errorf("WORKER_ID is required")
	}

	if c.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}

	if c.StreamKey == "" {
		return fmt.Errorf("STREAM_KEY is required")
	}

	if c.ConsumerGroup == "" {
		return fmt.Errorf("CONSUMER_GROUP is required")
	}

	if c.ResultStream == "" {
		return fmt.Errorf("RESULT_STREAM is required")
	}

	if c.LLMProvider == "" {
		return fmt.Errorf("LLM_PROVIDER is required")
	}

	// LLM_API_KEY is optional - only required when using LLM mode
	// It will be validated at runtime if LLM routing is attempted

	if c.LLMModel == "" {
		return fmt.Errorf("LLM_MODEL is required")
	}

	if c.LLMTimeout <= 0 {
		return fmt.Errorf("LLM_TIMEOUT must be positive")
	}

	if c.BlockTime <= 0 {
		return fmt.Errorf("BLOCK_TIME must be positive")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("MAX_RETRIES must be non-negative")
	}

	if c.HealthPort <= 0 || c.HealthPort > 65535 {
		return fmt.Errorf("HEALTH_PORT must be between 1 and 65535")
	}

	if !isValidLogLevel(c.LogLevel) {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	return nil
}

// isValidLogLevel checks if the log level is valid
func isValidLogLevel(level string) bool {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	return validLevels[level]
}

// RedisOptions returns Redis client options
func (c *Config) RedisOptions() map[string]interface{} {
	return map[string]interface{}{
		"addr":     c.RedisAddr,
		"password": c.RedisPassword,
		"db":       c.RedisDB,
	}
}

// LLMOptions returns LLM client options
func (c *Config) LLMOptions() map[string]interface{} {
	return map[string]interface{}{
		"provider": c.LLMProvider,
		"api_key":  c.LLMAPIKey,
		"model":    c.LLMModel,
		"timeout":  c.LLMTimeout,
	}
}

// String returns a string representation of the config (without sensitive data)
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{WorkerID=%s, RedisAddr=%s, RedisDB=%d, StreamKey=%s, ConsumerGroup=%s, "+
			"LLMProvider=%s, LLMModel=%s, CELEnabled=%v, HealthPort=%d, LogLevel=%s}",
		c.WorkerID,
		c.RedisAddr,
		c.RedisDB,
		c.StreamKey,
		c.ConsumerGroup,
		c.LLMProvider,
		c.LLMModel,
		c.CELEnabled,
		c.HealthPort,
		c.LogLevel,
	)
}
