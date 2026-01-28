.PHONY: build test proto run clean fmt lint help

# Default target
help:
	@echo "Available targets:"
	@echo "  build     - Build the server binary"
	@echo "  test      - Run all tests"
	@echo "  proto     - Generate protobuf code"
	@echo "  run       - Run the server"
	@echo "  clean     - Remove build artifacts"
	@echo "  fmt       - Format code"
	@echo "  lint      - Run linters"

# Build server binary
build:
	@echo "Building server..."
	@go build -o bin/server ./cmd/server

# Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.txt ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. api/proto/*.proto

# Run server locally
run:
	@echo "Running server..."
	@go run cmd/server/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.txt coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run ./... || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
