# DA Node Router

Router worker for the DA Orchestrator - routes graph execution flow using deterministic, LLM, or hybrid strategies.

## Overview

The DA Node Router is a horizontally scalable worker that:

- **Subscribes to Redis Streams** for routing work
- **Routes execution flow** based on state and rules
- **Three routing modes**: deterministic (CEL), LLM (semantic), hybrid (best of both)
- **Scales horizontally** - run multiple instances

## Architecture

```
┌─────────────────────────────────────────────────┐
│         Redis Streams (router.work)             │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│             Router Worker                        │
│  ┌──────────────────────────────────────────┐  │
│  │  Mode Detection & Routing                │  │
│  │  - Deterministic (CEL expressions)       │  │
│  │  - LLM (semantic understanding)          │  │
│  │  - Hybrid (CEL + LLM fallback)           │  │
│  └──────────────────────────────────────────┘  │
│                                                  │
│  ┌──────────┐  ┌─────────┐  ┌──────────────┐  │
│  │   CEL    │  │   LLM   │  │   Template   │  │
│  │Evaluator │  │  Client │  │    Engine    │  │
│  └──────────┘  └─────────┘  └──────────────┘  │
└─────────────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│       Redis Streams (router.decided)            │
└─────────────────────────────────────────────────┘
```

## Quick Start

### Using Docker

```bash
docker run -d \
  --name router-worker \
  -e REDIS_ADDR=redis:6379 \
  -e LLM_API_KEY=your-api-key \
  aescanero/dago-node-router:latest
```

### Building from Source

```bash
make deps
make build

export REDIS_ADDR=localhost:6379
export LLM_API_KEY=your-api-key
./router-worker
```

## Configuration

| Variable      | Default            | Description                  |
|---------------|--------------------|-----------------------------|
| `WORKER_ID`   | `router-1`         | Worker identifier           |
| `REDIS_ADDR`  | `localhost:6379`   | Redis server address        |
| `REDIS_PASS`  | (empty)            | Redis password              |
| `LLM_PROVIDER`| `anthropic`        | LLM provider                |
| `LLM_API_KEY` | (required for LLM) | LLM API key                 |
| `LLM_MODEL`   | `claude-sonnet-4-20250514` | LLM model        |
| `CEL_ENABLED` | `true`             | Enable CEL evaluator        |
| `LOG_LEVEL`   | `info`             | Log level                   |

## Routing Modes

### Deterministic Mode (CEL)

Fast, rule-based routing using CEL expressions:

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

### LLM Mode

Semantic routing with natural language understanding:

```json
{
  "mode": "llm",
  "prompt_template": "Classify: {{state.message}}\\nCategories: technical, billing, general",
  "routes": {
    "technical": "tech_support",
    "billing": "billing_dept",
    "general": "general_inquiry"
  },
  "fallback": "default_handler"
}
```

### Hybrid Mode

Best of both - fast rules + LLM fallback:

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

See [docs/ROUTING.md](docs/ROUTING.md) for detailed routing documentation.

## Scaling

Run multiple router workers:

```bash
docker-compose up -d --scale router-worker=3
```

Each worker processes routing decisions independently via Redis Streams consumer groups.

## Development

### Prerequisites

- Go 1.25.5+
- Redis 7.0+
- Docker (optional)

### Running Tests

```bash
make test
```

### Project Structure

```
dago-node-router/
├── cmd/router-worker/      # Main entry point
├── internal/
│   ├── router/             # Routing logic
│   ├── eval/               # CEL & template engines
│   ├── worker/             # Worker lifecycle
│   └── config/             # Configuration
├── deployments/docker/     # Docker files
└── docs/                   # Documentation
```

## Documentation

- [Routing Strategies](docs/ROUTING.md)
- [Worker Documentation](docs/README.md)
- [Changelog](docs/CHANGELOG.md)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- **Domain**: [disasterproject.com](https://disasterproject.com)
- **GitHub**: [github.com/aescanero/dago-node-router](https://github.com/aescanero/dago-node-router)
- **Docker Hub**: [aescanero/dago-node-router](https://hub.docker.com/r/aescanero/dago-node-router)
- **Dependencies**: [dago-libs](https://github.com/aescanero/dago-libs)
