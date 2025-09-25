# Makefile for ownarr

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary name
BINARY_NAME=ownarr
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directory
BUILD_DIR=./build

# Main package
MAIN_PACKAGE=./cmd/ownarr

.PHONY: all build test clean run deps fmt lint help install-tools vet

all: clean deps fmt lint vet test build

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PACKAGE)

## Build for Linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PACKAGE)

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## Lint code
lint:
	@echo "Linting code..."
	$(GOLINT) run ./...

## Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## Run with config file
run-config: build
	@echo "Running $(BINARY_NAME) with config..."
	$(BUILD_DIR)/$(BINARY_NAME) -config config.example.yaml

## Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t ownarr:latest .

## Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -v $(PWD)/config.example.yaml:/config.yaml -v /tmp:/data/media ownarr:latest -config /config.yaml

## Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-linux   - Build for Linux"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  benchmark     - Run benchmarks"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  vet           - Run go vet"
	@echo "  run           - Build and run the application"
	@echo "  run-config    - Build and run with example config"
	@echo "  install-tools - Install development tools"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Build and run Docker container"
	@echo "  help          - Show this help message"
