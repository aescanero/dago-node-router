package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/aescanero/dago-node-router/internal/config"
	"github.com/aescanero/dago-node-router/internal/router"
	"github.com/aescanero/dago-node-router/internal/worker"
	"encoding/json"

	"github.com/aescanero/dago-libs/pkg/domain/state"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version is set at build time
	Version = "dev"
	// BuildTime is set at build time
	BuildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("starting router worker",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
		zap.String("worker_id", cfg.WorkerID),
	)

	// Log configuration (without sensitive data)
	logger.Info("configuration loaded", zap.String("config", cfg.String()))

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	logger.Info("connected to redis", zap.String("addr", cfg.RedisAddr))

	// Initialize LLM client (optional for deterministic-only mode)
	var llmClient ports.LLMClient
	if cfg.LLMAPIKey != "" {
		llmClient, err = initLLMClient(cfg)
		if err != nil {
			logger.Warn("failed to initialize llm client (llm routing will not be available)",
				zap.Error(err),
			)
		} else {
			logger.Info("llm client initialized",
				zap.String("provider", cfg.LLMProvider),
				zap.String("model", cfg.LLMModel),
			)
		}
	} else {
		logger.Warn("llm api key not provided (llm routing will not be available)")
	}

	// Initialize event bus (Redis Streams implementation)
	eventBus := NewRedisEventBus(redisClient, logger)

	// Initialize state store (Redis JSON implementation)
	stateStore := NewRedisStateStore(redisClient, logger)

	// Initialize router
	routerInstance := router.NewRouter(llmClient, logger)
	logger.Info("router initialized")

	// Initialize worker
	w := worker.NewWorker(cfg, redisClient, routerInstance, eventBus, stateStore, logger)

	// Start worker
	if err := w.Start(); err != nil {
		logger.Fatal("failed to start worker", zap.Error(err))
	}

	// Start health server
	healthServer := worker.NewHealthServer(cfg.HealthPort, redisClient, logger)
	if err := healthServer.Start(); err != nil {
		logger.Fatal("failed to start health server", zap.Error(err))
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("router worker running, press Ctrl+C to stop")
	<-sigChan

	logger.Info("shutdown signal received, stopping worker")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop health server
	if err := healthServer.Stop(); err != nil {
		logger.Error("failed to stop health server", zap.Error(err))
	}

	// Stop worker
	if err := w.Stop(); err != nil {
		logger.Error("failed to stop worker", zap.Error(err))
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		logger.Error("failed to close redis connection", zap.Error(err))
	}

	select {
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timeout exceeded, forcing exit")
	default:
		logger.Info("worker stopped gracefully")
	}
}

// initLogger initializes the logger
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// initLLMClient initializes the LLM client based on provider
func initLLMClient(cfg *config.Config) (ports.LLMClient, error) {
	switch cfg.LLMProvider {
	case "anthropic":
		return newAnthropicClient(cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider: %s", cfg.LLMProvider)
	}
}

// anthropicClient implements ports.LLMClient for Anthropic Claude
// MVP: Stub implementation - TODO: Implement proper Anthropic SDK integration
type anthropicClient struct {
	apiKey  string
	model   string
	timeout time.Duration
	logger  *zap.Logger
}

// newAnthropicClient creates a new Anthropic client
func newAnthropicClient(apiKey, model string, timeout time.Duration) *anthropicClient {
	logger, _ := zap.NewProduction()
	return &anthropicClient{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		logger:  logger,
	}
}

// GenerateCompletion generates a completion from the LLM (compatibility method)
func (c *anthropicClient) GenerateCompletion(ctx context.Context, req interface{}) (interface{}, error) {
	llmReq, ok := req.(*domain.LLMRequest)
	if !ok {
		return nil, fmt.Errorf("expected *domain.LLMRequest, got %T", req)
	}

	c.logger.Warn("LLM client is stub implementation - returning mock response",
		zap.String("model", llmReq.Model))

	// Return mock response for MVP
	return &domain.LLMResponse{
		Content: "This is a stub LLM response. Implement Anthropic SDK integration.",
		Model:   llmReq.Model,
		Usage: domain.Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}, nil
}

// Complete performs a standard text completion
func (c *anthropicClient) Complete(ctx context.Context, req ports.CompletionRequest) (*ports.CompletionResponse, error) {
	c.logger.Warn("LLM Complete is stub implementation - returning mock response")
	return &ports.CompletionResponse{
		ID:    "stub-id",
		Model: req.Model,
		Message: ports.Message{
			Role:    "assistant",
			Content: "This is a stub response. Implement Anthropic SDK integration.",
		},
		FinishReason: "stop",
		Usage: ports.UsageInfo{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		CreatedAt: time.Now(),
	}, nil
}

// CompleteWithTools performs a completion with tool calling support
func (c *anthropicClient) CompleteWithTools(ctx context.Context, req ports.CompletionRequest, tools []ports.Tool) (*ports.CompletionResponse, error) {
	c.logger.Warn("LLM CompleteWithTools is stub implementation")
	return c.Complete(ctx, req)
}

// CompleteStructured performs a completion with guaranteed JSON schema conformance
func (c *anthropicClient) CompleteStructured(ctx context.Context, req ports.CompletionRequest, schema ports.JSONSchema) (*ports.StructuredResponse, error) {
	c.logger.Warn("LLM CompleteStructured is stub implementation - returning mock response")
	return &ports.StructuredResponse{
		Data: map[string]interface{}{
			"message": "This is a stub structured response. Implement Anthropic SDK integration.",
		},
		Usage: ports.UsageInfo{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		CreatedAt: time.Now(),
	}, nil
}

// RedisEventBus implements ports.EventBus using Redis Streams
type RedisEventBus struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisEventBus creates a new Redis event bus
func NewRedisEventBus(client *redis.Client, logger *zap.Logger) *RedisEventBus {
	return &RedisEventBus{
		client: client,
		logger: logger,
	}
}

// Publish publishes an event to a topic
func (e *RedisEventBus) Publish(ctx context.Context, topic string, event ports.Event) error {
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Redis stream
	_, err = e.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Subscribe registers a handler for events on a topic
func (e *RedisEventBus) Subscribe(ctx context.Context, topic string, handler ports.EventHandler) error {
	// Not implemented for router worker (stub)
	e.logger.Warn("Subscribe not implemented in router worker")
	return nil
}

// Unsubscribe removes a subscription from a topic
func (e *RedisEventBus) Unsubscribe(ctx context.Context, topic string) error {
	// Not implemented for router worker (stub)
	e.logger.Warn("Unsubscribe not implemented in router worker")
	return nil
}

// Close closes the event bus (no-op for Redis implementation)
func (e *RedisEventBus) Close() error {
	return nil
}

// RedisStateStore implements ports.StateStorage using Redis JSON
type RedisStateStore struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisStateStore creates a new Redis state store
func NewRedisStateStore(client *redis.Client, logger *zap.Logger) *RedisStateStore {
	return &RedisStateStore{
		client: client,
		logger: logger,
	}
}

// Save saves graph state
func (s *RedisStateStore) Save(ctx context.Context, executionID string, st state.State) error {
	key := fmt.Sprintf("graph:state:%s", executionID)

	// Marshal state to JSON
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Save to Redis
	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// Load loads graph state
func (s *RedisStateStore) Load(ctx context.Context, executionID string) (state.State, error) {
	key := fmt.Sprintf("graph:state:%s", executionID)

	// Get state from Redis
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("state not found for execution %s", executionID)
		}
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Unmarshal JSON to state.State (which is map[string]interface{})
	var st state.State
	if err := json.Unmarshal([]byte(data), &st); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return st, nil
}

// Delete deletes graph state
func (s *RedisStateStore) Delete(ctx context.Context, executionID string) error {
	key := fmt.Sprintf("graph:state:%s", executionID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// Exists checks if state exists for an execution
func (s *RedisStateStore) Exists(ctx context.Context, executionID string) (bool, error) {
	key := fmt.Sprintf("graph:state:%s", executionID)

	result, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// SetTTL sets a time-to-live for state data
func (s *RedisStateStore) SetTTL(ctx context.Context, executionID string, ttl time.Duration) error {
	key := fmt.Sprintf("graph:state:%s", executionID)

	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	return nil
}

// List returns all execution IDs that have stored state
func (s *RedisStateStore) List(ctx context.Context) ([]string, error) {
	keys, err := s.client.Keys(ctx, "graph:state:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	// Extract execution IDs from keys
	executionIDs := make([]string, 0, len(keys))
	prefix := "graph:state:"
	for _, key := range keys {
		if len(key) > len(prefix) {
			executionIDs = append(executionIDs, key[len(prefix):])
		}
	}

	return executionIDs, nil
}

// SaveState persists graph state (compatibility method)
func (s *RedisStateStore) SaveState(ctx context.Context, st interface{}) error {
	// Extract execution ID from state
	stateMap, ok := st.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map[string]interface{}, got %T", st)
	}

	executionID, ok := stateMap["graph_id"].(string)
	if !ok {
		executionID, ok = stateMap["execution_id"].(string)
		if !ok {
			return fmt.Errorf("state missing graph_id or execution_id field")
		}
	}

	return s.Save(ctx, executionID, state.State(stateMap))
}

// GetState retrieves graph state (compatibility method)
func (s *RedisStateStore) GetState(ctx context.Context, graphID string) (interface{}, error) {
	return s.Load(ctx, graphID)
}
