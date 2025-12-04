# DA Node Router - Architecture & Internals

## Overview

The DA Node Router is a horizontally scalable worker that processes routing decisions for graph execution in the DA Orchestrator. It subscribes to Redis Streams for work, evaluates routing rules, and publishes decisions.

## Architecture

### Components

```
┌─────────────────────────────────────────────────┐
│             Router Worker                        │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │         Worker (internal/worker)           │ │
│  │  - Subscribe to redis streams              │ │
│  │  - Process routing requests                │ │
│  │  - Publish routing decisions               │ │
│  │  - Health checks                           │ │
│  └────────────┬───────────────────────────────┘ │
│               │                                  │
│               ▼                                  │
│  ┌────────────────────────────────────────────┐ │
│  │         Router (internal/router)           │ │
│  │  - Mode detection                          │ │
│  │  - Strategy selection                      │ │
│  │  - Route resolution                        │ │
│  └────┬───────┬───────┬──────────────────────┘ │
│       │       │       │                          │
│       ▼       ▼       ▼                          │
│  ┌─────┐ ┌─────┐ ┌───────┐                     │
│  │ CEL │ │ LLM │ │Hybrid │                     │
│  └─────┘ └─────┘ └───────┘                     │
│     │       │         │                          │
│     ▼       ▼         ▼                          │
│  ┌──────────────────────────────────────────┐  │
│  │    Evaluation Engines (internal/eval)    │  │
│  │  - CEL evaluator                         │  │
│  │  - Template engine (Handlebars)          │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### Worker Layer (`internal/worker`)
- **Redis Streams Integration**: Subscribe to `router.work` stream
- **Work Processing**: Process routing requests from orchestrator
- **Result Publishing**: Publish to `router.decided` stream
- **Health Monitoring**: HTTP health endpoint for Kubernetes
- **Graceful Shutdown**: Clean resource cleanup

#### Router Layer (`internal/router`)
- **Mode Detection**: Automatically detect routing mode from config
- **Strategy Execution**: Execute deterministic, LLM, or hybrid routing
- **Route Resolution**: Map routing results to target nodes
- **Fallback Handling**: Default routes when no match found

#### Evaluation Layer (`internal/eval`)
- **CEL Evaluator**: Fast rule-based evaluation
- **Template Engine**: Handlebars template rendering for LLM prompts
- **State Access**: Access to graph state for decisions

## Routing Modes

### 1. Deterministic Mode (CEL)

Uses Common Expression Language for fast, rule-based routing.

**When to use:**
- Clear, objective routing criteria
- High performance requirements
- Predictable routing logic

**Example configuration:**
```json
{
  "mode": "deterministic",
  "rules": [
    {
      "condition": "state.priority == 'high'",
      "target": "urgent_handler"
    },
    {
      "condition": "state.score > 0.8",
      "target": "premium_flow"
    }
  ],
  "fallback": "default_handler"
}
```

**Implementation:**
- Evaluates rules in order
- First matching rule wins
- Falls back to default if no match
- No LLM calls required

### 2. LLM Mode

Uses semantic understanding for flexible routing decisions.

**When to use:**
- Natural language classification
- Semantic understanding required
- Ambiguous routing criteria
- Intent-based routing

**Example configuration:**
```json
{
  "mode": "llm",
  "prompt_template": "Classify: {{state.message}}\nCategories: technical, billing, general",
  "routes": {
    "technical": "tech_support",
    "billing": "billing_dept",
    "general": "general_inquiry"
  },
  "fallback": "default_handler"
}
```

**Implementation:**
- Constructs prompt using template engine
- Calls LLM for classification
- Maps LLM response to target node
- Falls back if LLM response doesn't match

### 3. Hybrid Mode

Combines fast CEL rules with LLM fallback.

**When to use:**
- Optimize common cases with rules
- Handle edge cases with LLM
- Balance performance and flexibility

**Example configuration:**
```json
{
  "mode": "hybrid",
  "fast_rules": [
    {
      "condition": "state.message.contains('refund')",
      "target": "refund_flow"
    }
  ],
  "llm_fallback": {
    "prompt_template": "...",
    "routes": {...}
  },
  "fallback": "default_handler"
}
```

**Implementation:**
1. Try CEL rules first (fast path)
2. If no match, use LLM (slow path)
3. Fall back to default if neither matches

## Data Flow

### Routing Request Flow

```
1. Orchestrator publishes to router.work
   ↓
2. Router worker reads from stream
   ↓
3. Load graph state from Redis
   ↓
4. Detect routing mode from node config
   ↓
5. Execute routing strategy:
   - Deterministic: Evaluate CEL rules
   - LLM: Construct prompt, call LLM, parse response
   - Hybrid: Try rules, fallback to LLM
   ↓
6. Resolve target node ID
   ↓
7. Publish decision to router.decided
   {
     "execution_id": "...",
     "node_id": "...",
     "target_node": "next_node_id",
     "reasoning": "..."
   }
   ↓
8. Acknowledge stream message
```

### State Access

Routers have read-only access to graph state:

```go
type GraphState struct {
    ExecutionID string
    GraphID     string
    Status      ExecutionStatus
    Variables   map[string]interface{}
    NodeStates  map[string]*NodeState
    // ...
}
```

Available in CEL expressions as `state.*` and template variables as `{{state.*}}`.

## Configuration

Environment variables:

| Variable       | Default                 | Description               |
|----------------|-------------------------|---------------------------|
| `WORKER_ID`    | `router-1`              | Worker identifier         |
| `REDIS_ADDR`   | `localhost:6379`        | Redis server address      |
| `REDIS_PASS`   | (empty)                 | Redis password            |
| `LLM_PROVIDER` | `anthropic`             | LLM provider              |
| `LLM_API_KEY`  | (required for LLM mode) | LLM API key               |
| `LLM_MODEL`    | `claude-sonnet-4-20250514` | LLM model          |
| `CEL_ENABLED`  | `true`                  | Enable CEL evaluator      |
| `LOG_LEVEL`    | `info`                  | Log level                 |
| `HEALTH_PORT`  | `8082`                  | Health check port         |

## Scaling

### Horizontal Scaling

Run multiple router workers:

```bash
docker run -d -e WORKER_ID=router-1 aescanero/dago-node-router
docker run -d -e WORKER_ID=router-2 aescanero/dago-node-router
docker run -d -e WORKER_ID=router-3 aescanero/dago-node-router
```

### Load Distribution

Redis Streams consumer groups automatically distribute work:
- Round-robin distribution
- Each message processed once
- Automatic redelivery on failure
- Pending message tracking

### Performance Characteristics

**Deterministic Mode:**
- Throughput: 1000+ routes/sec per worker
- Latency: < 10ms p99
- No external dependencies (LLM)

**LLM Mode:**
- Throughput: 10-50 routes/sec per worker (LLM dependent)
- Latency: 100-500ms p99
- Requires LLM API availability

**Hybrid Mode:**
- Fast path: Same as deterministic
- Slow path: Same as LLM
- Optimize fast_rules to maximize fast path hits

## Error Handling

### Transient Errors
- Redis connection errors → retry with backoff
- LLM timeout → retry once, then fallback
- Parse errors → fallback route

### Permanent Errors
- Invalid CEL syntax → log error, use fallback
- Invalid config → reject at validation
- Missing fallback → error to orchestrator

### Graceful Degradation
- LLM unavailable → use fallback route
- CEL evaluation error → try LLM (hybrid mode)
- All strategies fail → error to orchestrator

## Monitoring

### Health Checks

HTTP endpoint on `:8082`:
- `GET /health` - Overall health
- `GET /ready` - Readiness probe

### Metrics

(Future: Prometheus metrics)
- Routes processed
- Route resolution time
- Mode distribution
- Error rates
- LLM call latency

### Logging

Structured logging with Zap:
- Route decisions with reasoning
- Mode selection
- Error conditions
- Performance metrics

## Development

### Running Locally

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Set environment
export REDIS_ADDR=localhost:6379
export LLM_API_KEY=your-key

# Build and run
make run-local
```

### Testing

```bash
# Unit tests
make test

# Integration tests (requires Redis)
go test ./tests/integration/...

# Test specific mode
go test ./internal/router -run TestDeterministicRouting
```

### Adding New Routing Strategy

1. Implement strategy in `internal/router/`:
   ```go
   func (r *Router) routeCustom(ctx context.Context, ...) (string, error)
   ```

2. Add mode detection in `internal/router/router.go`

3. Add configuration parsing in `internal/config/config.go`

4. Add tests in `internal/router/*_test.go`

## Best Practices

### Deterministic Mode
- Keep rules simple and fast
- Order rules by specificity
- Use appropriate data types in CEL
- Test edge cases thoroughly

### LLM Mode
- Craft clear, concise prompts
- Provide examples in prompt
- Validate LLM responses
- Set appropriate timeouts

### Hybrid Mode
- Put common cases in fast_rules
- Keep fast_rules count reasonable (< 10)
- Test fast path coverage
- Monitor slow path usage

### General
- Always provide fallback route
- Log routing decisions with reasoning
- Monitor routing accuracy
- Review and optimize rules regularly

## Security

### Input Validation
- Validate all node configurations
- Sanitize state variables in prompts
- Limit CEL expression complexity

### LLM Security
- No sensitive data in prompts
- Validate LLM responses before use
- Rate limiting on LLM calls

### Redis Security
- Use Redis AUTH
- TLS for production
- Network isolation

## Future Enhancements

- [ ] A/B testing for routing strategies
- [ ] Machine learning-based routing
- [ ] Route caching for repeated patterns
- [ ] Advanced analytics and visualization
- [ ] Multi-LLM provider support
- [ ] Custom function support in CEL
- [ ] Routing strategy composition
