.PHONY: help build test test-coverage test-ci bench bench-compare vet lint fmt fmt-check clean deps proto run check

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
BINARY_NAME=server
BINARY_PATH=bin/$(BINARY_NAME)

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	@$(GOMOD) tidy

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOVET) ./...

fmt: ## Format code
	@echo "Formatting code..."
	@$(GOFMT) ./...

fmt-check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt' to fix." && gofmt -l . && exit 1)

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run --timeout=5m

test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@$(GOCMD) tool cover -func=coverage.txt | tail -1

test-ci: ## Run tests for CI (with coverage)
	@echo "Running tests for CI..."
	@$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem -run=^$$ ./...

bench-compare: ## Run benchmarks and save results for comparison
	@echo "Running benchmarks and saving results..."
	@$(GOTEST) -bench=. -benchmem -run=^$$ ./... | tee bench-new.txt
	@echo "Results saved to bench-new.txt"
	@echo "To compare with previous results, run: benchstat bench-old.txt bench-new.txt"

check: fmt-check vet lint test ## Run all checks (fmt-check, vet, lint, test)

build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@$(GOBUILD) -o $(BINARY_PATH) ./cmd/server

run: ## Run the server
	@echo "Running server..."
	@$(GOCMD) run cmd/server/main.go

proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@which protoc > /dev/null || (echo "protoc not installed. Install from https://grpc.io/docs/protoc-installation/" && exit 1)
	@protoc --go_out=. --go-grpc_out=. api/proto/*.proto

clean: ## Clean build artifacts and coverage files
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.txt coverage.html

.PHONY: all
all: deps vet lint test build ## Run all steps (deps, vet, lint, test, build)
