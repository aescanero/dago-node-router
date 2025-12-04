.PHONY: help deps test lint fmt clean build docker-build docker-push run-local release

# Variables
BINARY_NAME=router-worker
DOCKER_IMAGE=aescanero/dago-node-router
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod download
	go mod verify

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.txt
	rm -f $(BINARY_NAME)

build: ## Build binary
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/router-worker

build-linux: ## Build binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/router-worker

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest -f deployments/docker/Dockerfile .

docker-push: ## Push Docker image
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

run-local: build ## Build and run locally
	./$(BINARY_NAME)

release: ## Create a new release
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
