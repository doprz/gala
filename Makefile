# Gala - Git Author Line Analyzer
# Makefile for building and managing the project

# Build variables
BINARY_NAME=gala
BINARY_UNIX=$(BINARY_NAME)_unix
VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

.PHONY: all build clean test deps install uninstall help

# Default target
all: deps test build

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 -v
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 -v

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 -v
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 -v

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe -v

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf bin/

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Install the binary to GOPATH/bin
install: build
	$(GOCMD) install $(LDFLAGS)

# Uninstall the binary from GOPATH/bin
uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Run the application
run: build
	./$(BINARY_NAME)

# Run with example arguments
run-example: build
	./$(BINARY_NAME) --verbose

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Security scan (requires gosec)
security:
	gosec ./...

# Generate documentation
docs:
	$(GOCMD) doc -all

# Development setup
dev-setup:
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec

# Create release archives
release: clean build-all
	mkdir -p release
	tar -czf release/$(BINARY_NAME)-v$(VERSION)-linux-amd64.tar.gz -C bin $(BINARY_NAME)-linux-amd64
	tar -czf release/$(BINARY_NAME)-v$(VERSION)-linux-arm64.tar.gz -C bin $(BINARY_NAME)-linux-arm64
	tar -czf release/$(BINARY_NAME)-v$(VERSION)-darwin-amd64.tar.gz -C bin $(BINARY_NAME)-darwin-amd64
	tar -czf release/$(BINARY_NAME)-v$(VERSION)-darwin-arm64.tar.gz -C bin $(BINARY_NAME)-darwin-arm64
	zip -j release/$(BINARY_NAME)-v$(VERSION)-windows-amd64.zip bin/$(BINARY_NAME)-windows-amd64.exe

# Docker build (optional)
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all supported platforms"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy dependencies"
	@echo "  install      - Install binary to GOPATH/bin"
	@echo "  uninstall    - Remove binary from GOPATH/bin"
	@echo "  run          - Build and run the application"
	@echo "  run-example  - Build and run with example arguments"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter (requires golangci-lint)"
	@echo "  security     - Run security scan (requires gosec)"
	@echo "  docs         - Generate documentation"
	@echo "  dev-setup    - Install development tools"
	@echo "  release      - Create release archives for all platforms"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help message"
