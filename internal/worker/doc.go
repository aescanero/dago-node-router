// Package worker implements the router worker lifecycle and Redis Streams integration.
//
// The worker subscribes to Redis Streams for routing work, processes routing decisions,
// and publishes results back to the orchestrator.
//
// Example usage:
//
//	cfg, _ := config.Load()
//	redisClient := redis.NewClient(&redis.Options{...})
//	router := router.NewRouter(llmClient, logger)
//
//	worker := worker.NewWorker(cfg, redisClient, router, eventBus, stateStore, logger)
//	if err := worker.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer worker.Stop()
//
// The worker handles:
//   - Redis Streams subscription and consumer group management
//   - Routing request processing
//   - Routing decision publishing
//   - Error handling and reporting
//   - Graceful shutdown
//
// Health checks are provided via a separate HTTP server:
//
//	healthServer := worker.NewHealthServer(8082, redisClient, logger)
//	healthServer.Start()
//	defer healthServer.Stop()
package worker
