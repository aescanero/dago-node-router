#!/bin/bash

set -e

# Script to run the router worker locally for development

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "=== DA Node Router - Local Development ==="
echo

# Check if Redis is running
echo "Checking Redis connection..."
if ! redis-cli -h "${REDIS_ADDR:-localhost}" -p 6379 ping > /dev/null 2>&1; then
    echo "Error: Redis is not running or not accessible"
    echo "Please start Redis with: docker run -d -p 6379:6379 redis:7-alpine"
    exit 1
fi
echo "✓ Redis is running"
echo

# Check environment variables
echo "Checking environment variables..."
if [ -z "$LLM_API_KEY" ]; then
    echo "Warning: LLM_API_KEY is not set (LLM routing will not be available)"
    echo "Set it with: export LLM_API_KEY=your-key"
else
    echo "✓ LLM_API_KEY is set"
fi
echo

# Set default environment variables if not set
export WORKER_ID="${WORKER_ID:-router-local-1}"
export REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"
export LOG_LEVEL="${LOG_LEVEL:-debug}"
export HEALTH_PORT="${HEALTH_PORT:-8082}"

echo "Configuration:"
echo "  WORKER_ID: $WORKER_ID"
echo "  REDIS_ADDR: $REDIS_ADDR"
echo "  LOG_LEVEL: $LOG_LEVEL"
echo "  HEALTH_PORT: $HEALTH_PORT"
echo

# Build the binary
echo "Building router worker..."
./scripts/build.sh
echo

# Run the binary
echo "Starting router worker..."
echo "Press Ctrl+C to stop"
echo
./router-worker
