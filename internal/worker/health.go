package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// HealthServer provides HTTP health check endpoints
type HealthServer struct {
	port        int
	redisClient *redis.Client
	logger      *zap.Logger
	server      *http.Server
}

// NewHealthServer creates a new health server
func NewHealthServer(port int, redisClient *redis.Client, logger *zap.Logger) *HealthServer {
	return &HealthServer{
		port:        port,
		redisClient: redisClient,
		logger:      logger,
	}
}

// Start starts the health check server
func (hs *HealthServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", hs.handleHealth)
	mux.HandleFunc("/ready", hs.handleReady)

	hs.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", hs.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	hs.logger.Info("starting health server", zap.Int("port", hs.port))

	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logger.Error("health server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the health check server
func (hs *HealthServer) Stop() error {
	if hs.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hs.logger.Info("stopping health server")
	return hs.server.Shutdown(ctx)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// handleHealth handles the /health endpoint
func (hs *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)

	// Check Redis connection
	if err := hs.redisClient.Ping(ctx).Err(); err != nil {
		checks["redis"] = fmt.Sprintf("unhealthy: %v", err)
		hs.respondJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "unhealthy",
			Checks: checks,
		})
		return
	}
	checks["redis"] = "healthy"

	// All checks passed
	hs.respondJSON(w, http.StatusOK, HealthResponse{
		Status: "healthy",
		Checks: checks,
	})
}

// handleReady handles the /ready endpoint
func (hs *HealthServer) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check if Redis is ready
	if err := hs.redisClient.Ping(ctx).Err(); err != nil {
		hs.respondJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status: "not ready",
		})
		return
	}

	// Worker is ready
	hs.respondJSON(w, http.StatusOK, HealthResponse{
		Status: "ready",
	})
}

// respondJSON writes a JSON response
func (hs *HealthServer) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		hs.logger.Error("failed to encode response", zap.Error(err))
	}
}
