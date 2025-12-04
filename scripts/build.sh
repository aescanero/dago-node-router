#!/bin/bash

set -e

# Script to build the router worker binary

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Get version and build time
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

BINARY_NAME="router-worker"

echo "Building router worker..."
echo "  Version: $VERSION"
echo "  Build time: $BUILD_TIME"

# Check if building for all platforms
if [ "$1" = "all" ]; then
    echo "Building for all platforms..."

    mkdir -p bin

    # Linux amd64
    echo "Building for linux/amd64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-linux-amd64" ./cmd/router-worker

    # Linux arm64
    echo "Building for linux/arm64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-linux-arm64" ./cmd/router-worker

    # macOS amd64
    echo "Building for darwin/amd64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-darwin-amd64" ./cmd/router-worker

    # macOS arm64
    echo "Building for darwin/arm64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-darwin-arm64" ./cmd/router-worker

    # Windows amd64
    echo "Building for windows/amd64..."
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-windows-amd64.exe" ./cmd/router-worker

    # Windows arm64
    echo "Building for windows/arm64..."
    CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "bin/${BINARY_NAME}-windows-arm64.exe" ./cmd/router-worker

    echo "All builds completed successfully!"
    ls -lh bin/
else
    # Build for current platform
    echo "Building for current platform..."
    CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${BINARY_NAME}" ./cmd/router-worker

    echo "Build completed successfully!"
    ls -lh "${BINARY_NAME}"
fi
