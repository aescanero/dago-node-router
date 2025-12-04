# Multi-stage build for DA Node Router

# Stage 1: Build
FROM golang:1.25.5-alpine AS builder

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o router-worker \
    ./cmd/router-worker

# Stage 2: Runtime
FROM alpine:3.23

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 router && \
    adduser -D -u 1000 -G router router

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/router-worker .

# Change ownership
RUN chown -R router:router /app

# Switch to non-root user
USER router

# Expose health check port
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/router-worker"]
