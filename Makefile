# Makefile
.PHONY: build run test clean install dev docker

# Variables
BINARY_NAME=jprq
BINARY_PATH=./bin/$(BINARY_NAME)
SERVER_BINARY=jprq-server
SERVER_PATH=./bin/$(SERVER_BINARY)

# Build flags
BUILD_FLAGS=-ldflags "-X main.version=$(shell git describe --tags --always)"

# Default target
all: build

# Build binaries
build:
	@echo "🔨 Building binaries..."
	@mkdir -p bin
	go build $(BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/client
	go build $(BUILD_FLAGS) -o $(SERVER_PATH) ./cmd/server
	@echo "✅ Build complete!"

# Run server in development mode
dev-server:
	@echo "🚀 Starting development server..."
	air -c .air.server.toml

# Run client in development mode  
dev-client:
	@echo "🚀 Starting development client..."
	air -c .air.client.toml

# Install binaries to system
install: build
	@echo "📦 Installing binaries..."
	sudo cp $(BINARY_PATH) /usr/local/bin/
	sudo cp $(SERVER_PATH) /usr/local/bin/
	@echo "✅ Installation complete!"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -rf certs/
	go clean

# Generate SSL certificates for development
dev-certs:
	@echo "🔐 Generating development certificates..."
	@mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes -subj "/CN=localhost"

# Docker build
docker:
	@echo "🐳 Building Docker image..."
	docker build -t jprq:latest .

# Release build for multiple platforms
release:
	@echo "📦 Building release binaries..."
	@mkdir -p releases
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(BINARY_NAME)-linux-amd64 ./cmd/client
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(SERVER_BINARY)-linux-amd64 ./cmd/server
	
	# Linux ARM64  
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o releases/$(BINARY_NAME)-linux-arm64 ./cmd/client
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o releases/$(SERVER_BINARY)-linux-arm64 ./cmd/server
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(BINARY_NAME)-darwin-amd64 ./cmd/client
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(SERVER_BINARY)-darwin-amd64 ./cmd/server
	
	# macOS ARM64 (M1/M2)
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o releases/$(BINARY_NAME)-darwin-arm64 ./cmd/client  
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o releases/$(SERVER_BINARY)-darwin-arm64 ./cmd/server
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(BINARY_NAME)-windows-amd64.exe ./cmd/client
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o releases/$(SERVER_BINARY)-windows-amd64.exe ./cmd/server
	
	@echo "✅ Release build complete!"

# Setup development environment
setup:
	@echo "🔧 Setting up development environment..."
	go mod tidy
	go install github.com/cosmtrek/air@latest
	@echo "✅ Development environment ready!"

# Show help
help:
	@echo "Available commands:"
	@echo "  build      - Build binaries"
	@echo "  dev-server - Run server in development mode"
	@echo "  dev-client - Run client in development mode"
	@echo "  install    - Install binaries to system"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  dev-certs  - Generate development SSL certificates"
	@echo "  docker     - Build Docker image"
	@echo "  release    - Build release binaries for all platforms"
	@echo "  setup      - Setup development environment"
	@echo "  help       - Show this help message"