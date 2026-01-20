.PHONY: build clean test install dev

BINARY_NAME=rcm
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/AhmedAburady/rcm-go/internal/cmd.Version=$(VERSION) -X github.com/AhmedAburady/rcm-go/internal/cmd.Commit=$(COMMIT) -X github.com/AhmedAburady/rcm-go/internal/cmd.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Build binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/rcm

# Development build (no ldflags, faster)
dev:
	go build -o bin/$(BINARY_NAME) ./cmd/rcm

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/rcm

# Run tests
test:
	go test -v ./...

# Run tests with coverage
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Cross-compilation targets
build-all: build-linux build-darwin build-darwin-arm64

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/rcm

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/rcm

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/rcm

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  dev            - Quick development build"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  test           - Run tests"
	@echo "  cover          - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  tidy           - Tidy go modules"
	@echo "  fmt            - Format code"
	@echo "  build-all      - Build for all platforms"
	@echo "  build-linux    - Build for Linux"
	@echo "  build-darwin   - Build for macOS (Intel)"
	@echo "  build-darwin-arm64 - Build for macOS (Apple Silicon)"
