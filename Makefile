# Makefile for backup-pro

# Binary name
BINARY_NAME=backup-pro

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Version information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Linker flags to inject version information
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Platforms for cross-compilation
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test test-verbose fmt vet deps deps-update help \
        build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 \
        build-windows-amd64 build-all

# Default target
all: clean fmt vet test build

# Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@echo "Dependencies downloaded and verified"

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Cross-platform builds
build-linux-amd64:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build-linux-arm64:
	@echo "Building for Linux arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

build-darwin-amd64:
	@echo "Building for macOS amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

build-darwin-arm64:
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

build-windows-amd64:
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build for all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo "All platform builds complete"

# Display help information
help:
	@echo "Available targets:"
	@echo "  make build              - Build binary for current platform"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make test-verbose       - Run tests with coverage report"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"
	@echo "  make deps               - Download and verify dependencies"
	@echo "  make deps-update        - Update dependencies (go mod tidy)"
	@echo ""
	@echo "Cross-platform builds:"
	@echo "  make build-linux-amd64  - Build for Linux x86_64"
	@echo "  make build-linux-arm64  - Build for Linux ARM64"
	@echo "  make build-darwin-amd64 - Build for macOS Intel"
	@echo "  make build-darwin-arm64 - Build for macOS Apple Silicon"
	@echo "  make build-windows-amd64- Build for Windows x86_64"
	@echo "  make build-all          - Build for all platforms"
	@echo ""
	@echo "  make all                - Run clean, fmt, vet, test, and build"
	@echo "  make help               - Show this help message"
