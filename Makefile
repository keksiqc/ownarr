.PHONY: build lint fmt clean run help

# Build the binary
build:
	go build -o bin/ownarr cmd/ownarr/main.go

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run the application
run:
	go run cmd/ownarr/main.go

# Show help
help:
	@echo "Available commands:"
	@echo "  build  - Build the binary"
	@echo "  fmt    - Format code"
	@echo "  clean  - Clean build artifacts"
	@echo "  run    - Run the application"
	@echo "  help   - Show this help"
