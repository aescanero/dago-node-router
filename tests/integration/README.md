# Integration Tests

Integration tests for the DA Node Router.

## Running Integration Tests

Integration tests require Redis to be running:

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Run integration tests
go test -v ./tests/integration/...
```

## Test Coverage

Integration tests cover:
- Redis Streams subscription and message processing
- Router logic with real CEL evaluator
- Worker lifecycle (start, stop, graceful shutdown)
- Health check endpoints
- Error handling and recovery

## Writing Integration Tests

Example integration test:

```go
package integration

import (
    "context"
    "testing"
    "time"

    "github.com/aescanero/dago-node-router/internal/router"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRouterIntegration(t *testing.T) {
    // Setup Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    // Test routing logic
    // ...
}
```

## Test Data

Test fixtures are located in `testdata/` directory.
