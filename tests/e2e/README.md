# End-to-End Tests

End-to-end tests for the DA Node Router.

## Running E2E Tests

E2E tests require the full environment (Redis, router worker):

```bash
# Start Redis
docker run -d -p 6379:6379 --name redis redis:7-alpine

# Start router worker
export REDIS_ADDR=localhost:6379
export LLM_API_KEY=your-key
./router-worker &

# Run E2E tests
go test -v ./tests/e2e/...

# Cleanup
killall router-worker
docker stop redis && docker rm redis
```

## Test Coverage

E2E tests cover:
- Complete routing workflows (deterministic, LLM, hybrid)
- Multi-worker scenarios
- Performance and load testing
- Failure scenarios and recovery
- Integration with orchestrator

## Writing E2E Tests

Example E2E test:

```go
package e2e

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDeterministicRoutingE2E(t *testing.T) {
    // Setup Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    // Publish routing request
    request := map[string]interface{}{
        "execution_id": "test-123",
        "node_id": "router-1",
        "config": map[string]interface{}{
            "mode": "deterministic",
            "rules": []map[string]string{
                {"condition": "state.priority == 'high'", "target": "urgent"},
            },
            "fallback": "default",
        },
    }

    data, _ := json.Marshal(request)
    client.XAdd(context.Background(), &redis.XAddArgs{
        Stream: "router.work",
        Values: map[string]interface{}{"data": string(data)},
    })

    // Wait for result
    time.Sleep(time.Second)

    // Verify result was published
    streams, err := client.XRead(context.Background(), &redis.XReadArgs{
        Streams: []string{"router.decided", "0"},
        Count:   1,
    }).Result()

    require.NoError(t, err)
    assert.NotEmpty(t, streams)
}
```

## Performance Tests

Performance tests are run with:

```bash
go test -v -run=TestPerformance ./tests/e2e/... -timeout=10m
```

Expected performance:
- Deterministic mode: 1000+ routes/sec
- LLM mode: 10-50 routes/sec
- Hybrid mode (70% fast path): 200+ routes/sec
