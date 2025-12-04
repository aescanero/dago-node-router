package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/aescanero/dago-node-router/internal/config"
	"github.com/aescanero/dago-node-router/internal/router"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Worker represents the router worker
type Worker struct {
	id            string
	config        *config.Config
	redisClient   *redis.Client
	router        *router.Router
	eventBus      ports.EventBus
	stateStore    ports.StateStorage
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	streamKey     string
	consumerGroup string
	resultStream  string
}

// NewWorker creates a new worker
func NewWorker(
	cfg *config.Config,
	redisClient *redis.Client,
	routerInstance *router.Router,
	eventBus ports.EventBus,
	stateStore ports.StateStorage,
	logger *zap.Logger,
) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		id:            cfg.WorkerID,
		config:        cfg,
		redisClient:   redisClient,
		router:        routerInstance,
		eventBus:      eventBus,
		stateStore:    stateStore,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		streamKey:     cfg.StreamKey,
		consumerGroup: cfg.ConsumerGroup,
		resultStream:  cfg.ResultStream,
	}
}

// Start starts the worker
func (w *Worker) Start() error {
	w.logger.Info("starting router worker",
		zap.String("worker_id", w.id),
		zap.String("stream_key", w.streamKey),
		zap.String("consumer_group", w.consumerGroup),
	)

	// Create consumer group if it doesn't exist
	if err := w.ensureConsumerGroup(); err != nil {
		return fmt.Errorf("failed to ensure consumer group: %w", err)
	}

	// Start processing work
	go w.processWork()

	w.logger.Info("router worker started", zap.String("worker_id", w.id))
	return nil
}

// Stop stops the worker gracefully
func (w *Worker) Stop() error {
	w.logger.Info("stopping router worker", zap.String("worker_id", w.id))

	// Cancel context to stop work processing
	w.cancel()

	// Wait a bit for in-flight work to complete
	time.Sleep(2 * time.Second)

	w.logger.Info("router worker stopped", zap.String("worker_id", w.id))
	return nil
}

// ensureConsumerGroup creates the consumer group if it doesn't exist
func (w *Worker) ensureConsumerGroup() error {
	// Try to create the group
	err := w.redisClient.XGroupCreateMkStream(w.ctx, w.streamKey, w.consumerGroup, "0").Err()
	if err != nil {
		// BUSYGROUP error means the group already exists, which is fine
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			w.logger.Debug("consumer group already exists",
				zap.String("group", w.consumerGroup),
			)
			return nil
		}
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	w.logger.Info("created consumer group",
		zap.String("group", w.consumerGroup),
		zap.String("stream", w.streamKey),
	)
	return nil
}

// processWork processes work from the Redis stream
func (w *Worker) processWork() {
	w.logger.Info("starting work processing loop")

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("work processing loop stopped")
			return
		default:
			// Read from stream
			streams, err := w.redisClient.XReadGroup(w.ctx, &redis.XReadGroupArgs{
				Group:    w.consumerGroup,
				Consumer: w.id,
				Streams:  []string{w.streamKey, ">"},
				Count:    1,
				Block:    w.config.BlockTime,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// No messages available, continue
					continue
				}
				w.logger.Error("failed to read from stream",
					zap.Error(err),
				)
				time.Sleep(time.Second)
				continue
			}

			// Process each message
			for _, stream := range streams {
				for _, message := range stream.Messages {
					w.handleMessage(message)
				}
			}
		}
	}
}

// handleMessage handles a single routing request message
func (w *Worker) handleMessage(message redis.XMessage) {
	messageID := message.ID
	w.logger.Info("processing routing request",
		zap.String("message_id", messageID),
	)

	// Parse the work request
	workRequest, err := w.parseWorkRequest(message.Values)
	if err != nil {
		w.logger.Error("failed to parse work request",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
		w.acknowledgeMessage(messageID)
		return
	}

	// Process the routing request
	if err := w.processRoutingRequest(workRequest); err != nil {
		w.logger.Error("failed to process routing request",
			zap.String("message_id", messageID),
			zap.String("execution_id", workRequest.ExecutionID),
			zap.Error(err),
		)
		// Publish error event
		w.publishError(workRequest, err)
	}

	// Acknowledge the message
	w.acknowledgeMessage(messageID)
}

// WorkRequest represents a routing work request
type WorkRequest struct {
	ExecutionID string                 `json:"execution_id"`
	NodeID      string                 `json:"node_id"`
	Config      map[string]interface{} `json:"config"`
}

// parseWorkRequest parses a work request from Redis message
func (w *Worker) parseWorkRequest(values map[string]interface{}) (*WorkRequest, error) {
	dataStr, ok := values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'data' field")
	}

	var request WorkRequest
	if err := json.Unmarshal([]byte(dataStr), &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal work request: %w", err)
	}

	return &request, nil
}

// processRoutingRequest processes a routing request
func (w *Worker) processRoutingRequest(request *WorkRequest) error {
	ctx := context.Background()

	// Load graph state from store
	stateData, err := w.stateStore.Load(ctx, request.ExecutionID)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Convert state.State (map) to domain.GraphState
	graphState, err := w.convertToGraphState(request.ExecutionID, stateData)
	if err != nil {
		return fmt.Errorf("failed to convert state: %w", err)
	}

	// Parse routing configuration
	nodeConfig, err := w.parseNodeConfig(request.Config)
	if err != nil {
		return fmt.Errorf("failed to parse node config: %w", err)
	}

	// Perform routing
	result, err := w.router.Route(ctx, graphState, nodeConfig)
	if err != nil {
		return fmt.Errorf("routing failed: %w", err)
	}

	// Publish routing decision
	if err := w.publishDecision(request, result); err != nil {
		return fmt.Errorf("failed to publish decision: %w", err)
	}

	return nil
}

// parseNodeConfig parses the node configuration into router.NodeConfig
func (w *Worker) parseNodeConfig(config map[string]interface{}) (*router.NodeConfig, error) {
	// Marshal and unmarshal to convert map to struct
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var nodeConfig router.NodeConfig
	if err := json.Unmarshal(data, &nodeConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &nodeConfig, nil
}

// publishDecision publishes the routing decision
func (w *Worker) publishDecision(request *WorkRequest, result *router.RoutingResult) error {
	decision := map[string]interface{}{
		"execution_id": request.ExecutionID,
		"node_id":      request.NodeID,
		"target_node":  result.TargetNode,
		"reasoning":    result.Reasoning,
		"mode":         result.Mode,
		"path_taken":   result.PathTaken,
		"timestamp":    time.Now().UTC(),
	}

	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("failed to marshal decision: %w", err)
	}

	// Publish to result stream
	_, err = w.redisClient.XAdd(w.ctx, &redis.XAddArgs{
		Stream: w.resultStream,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to publish to stream: %w", err)
	}

	w.logger.Info("published routing decision",
		zap.String("execution_id", request.ExecutionID),
		zap.String("target_node", result.TargetNode),
	)

	return nil
}

// publishError publishes an error event
func (w *Worker) publishError(request *WorkRequest, err error) {
	errorEvent := map[string]interface{}{
		"execution_id": request.ExecutionID,
		"node_id":      request.NodeID,
		"error":        err.Error(),
		"timestamp":    time.Now().UTC(),
	}

	data, marshalErr := json.Marshal(errorEvent)
	if marshalErr != nil {
		w.logger.Error("failed to marshal error event", zap.Error(marshalErr))
		return
	}

	// Publish error to a separate stream
	_, publishErr := w.redisClient.XAdd(w.ctx, &redis.XAddArgs{
		Stream: w.resultStream + ".errors",
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if publishErr != nil {
		w.logger.Error("failed to publish error event", zap.Error(publishErr))
	}
}

// acknowledgeMessage acknowledges a message from the stream
func (w *Worker) acknowledgeMessage(messageID string) {
	err := w.redisClient.XAck(w.ctx, w.streamKey, w.consumerGroup, messageID).Err()
	if err != nil {
		w.logger.Error("failed to acknowledge message",
			zap.String("message_id", messageID),
			zap.Error(err),
		)
	}
}

// convertToGraphState converts state.State to domain.GraphState
func (w *Worker) convertToGraphState(graphID string, stateData map[string]interface{}) (*domain.GraphState, error) {
	// Marshal the state data to JSON then unmarshal to GraphState
	data, err := json.Marshal(stateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	var graphState domain.GraphState
	if err := json.Unmarshal(data, &graphState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to GraphState: %w", err)
	}

	// Ensure GraphID is set
	if graphState.GraphID == "" {
		graphState.GraphID = graphID
	}

	return &graphState, nil
}
