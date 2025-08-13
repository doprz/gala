# Gala - Git Author Line Analyzer
# Makefile for building and managing the project

# Build variables
BINARY_NAME=gala
VERSION=1.0.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Directories
BIN_DIR=bin
RELEASE_DIR=release
COMPLETIONS_DIR=completions

.PHONY: all build clean test deps install uninstall help completions

# Default target
all: deps test build

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 -v
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 -v

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 -v
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 -v

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe -v

# Generate shell completions
completions: build
	mkdir -p $(COMPLETIONS_DIR)
	./$(BINARY_NAME) completion bash > $(COMPLETIONS_DIR)/$(BINARY_NAME).bash
	./$(BINARY_NAME) completion zsh > $(COMPLETIONS_DIR)/$(BINARY_NAME).zsh
	./$(BINARY_NAME) completion fish > $(COMPLETIONS_DIR)/$(BINARY_NAME).fish
	./$(BINARY_NAME) completion powershell > $(COMPLETIONS_DIR)/$(BINARY_NAME).ps1
	@echo "Shell completions generated in $(COMPLETIONS_DIR)/"

# Install completions (requires sudo for system-wide)
install-completions: completions
	@echo "Installing shell completions..."
	# Bash completion
	@if [ -d "/etc/bash_completion.d" ]; then \
		sudo cp $(COMPLETIONS_DIR)/$(BINARY_NAME).bash /etc/bash_completion.d/$(BINARY_NAME); \
		echo "Bash completion installed to /etc/bash_completion.d/"; \
	elif [ -d "/usr/local/etc/bash_completion.d" ]; then \
		sudo cp $(COMPLETIONS_DIR)/$(BINARY_NAME).bash /usr/local/etc/bash_completion.d/$(BINARY_NAME); \
		echo "Bash completion installed to /usr/local/etc/bash_completion.d/"; \
	fi
	# Zsh completion (user install)
	@if [ -n "$ZSH" ] && [ -d "$ZSH/completions" ]; then \
		cp $(COMPLETIONS_DIR)/$(BINARY_NAME).zsh $ZSH/completions/_$(BINARY_NAME); \
		echo "Zsh completion installed to $ZSH/completions/"; \
	fi
	# Fish completion (user install)
	@if [ -d "$HOME/.config/fish/completions" ]; then \
		cp $(COMPLETIONS_DIR)/$(BINARY_NAME).fish $HOME/.config/fish/completions/$(BINARY_NAME).fish; \
		echo "Fish completion installed to ~/.config/fish/completions/"; \
	fi

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BIN_DIR)/
	rm -rf $(RELEASE_DIR)/
	rm -rf $(COMPLETIONS_DIR)/

# Run tests
test:
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmark tests
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) verify

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Vendor dependencies
vendor:
	$(GOMOD) vendor

# Install the binary to GOPATH/bin
install: build
	$(GOCMD) install $(LDFLAGS)

# Uninstall the binary from GOPATH/bin
uninstall:
	rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)

# Run the application
run: build
	./$(BINARY_NAME)

# Run with example arguments
run-example: build
	./$(BINARY_NAME) --verbose --emoji

# Run with user analysis
run-user: build
	./$(BINARY_NAME) . "$(USER)" --verbose

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Fix linting issues
lint-fix:
	golangci-lint run --fix

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
release: clean build-all completions
	mkdir -p $(RELEASE_DIR)
	# Linux AMD64
	tar -czf $(RELEASE_DIR)/$(BINARY_NAME)-v$(VERSION)-linux-amd64.tar.gz \
		-C $(BIN_DIR) $(BINARY_NAME)-linux-amd64 \
		-C ../$(COMPLETIONS_DIR) $(BINARY_NAME).bash $(BINARY_NAME).zsh $(BINARY_NAME).fish \
		-C .. README.md LICENSE gala.yaml.example
	# Linux ARM64
	tar -czf $(RELEASE_DIR)/$(BINARY_NAME)-v$(VERSION)-linux-arm64.tar.gz \
		-C $(BIN_DIR) $(BINARY_NAME)-linux-arm64 \
		-C ../$(COMPLETIONS_DIR) $(BINARY_NAME).bash $(BINARY_NAME).zsh $(BINARY_NAME).fish \
		-C .. README.md LICENSE gala.yaml.example
	# macOS AMD64
	tar -czf $(RELEASE_DIR)/$(BINARY_NAME)-v$(VERSION)-darwin-amd64.tar.gz \
		-C $(BIN_DIR) $(BINARY_NAME)-darwin-amd64 \
		-C ../$(COMPLETIONS_DIR) $(BINARY_NAME).bash $(BINARY_NAME).zsh $(BINARY_NAME).fish \
		-C .. README.md LICENSE gala.yaml.example
	# macOS ARM64
	tar -czf $(RELEASE_DIR)/$(BINARY_NAME)-v$(VERSION)-darwin-arm64.tar.gz \
		-C $(BIN_DIR) $(BINARY_NAME)-darwin-arm64 \
		-C ../$(COMPLETIONS_DIR) $(BINARY_NAME).bash $(BINARY_NAME).zsh $(BINARY_NAME).fish \
		-C .. README.md LICENSE gala.yaml.example
	# Windows
	cd $(BIN_DIR) && zip -r ../$(RELEASE_DIR)/$(BINARY_NAME)-v$(VERSION)-windows-amd64.zip \
		$(BINARY_NAME)-windows-amd64.exe \
		../$(COMPLETIONS_DIR)/$(BINARY_NAME).ps1 \
		../README.md ../LICENSE ../gala.yaml.example
	@echo "Release archives created in $(RELEASE_DIR)/"

# Docker build
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Docker run
docker-run:
	docker run --rm -v $(PWD):/workspace $(BINARY_NAME):latest

# Nix build
nix-build:
	nix build

# Nix development shell
nix-shell:
	nix develop

# Check for updates
check-updates:
	$(GOCMD) list -u -m all

# Profile the application
profile: build
	./$(BINARY_NAME) --output plain > /dev/null &
	sleep 1
	go tool pprof http://localhost:6060/debug/pprof/profile

# Generate mock files (if using gomock)
generate:
	$(GOCMD) generate ./...

# Validate configuration
validate-config:
	@if [ -f "gala.yaml" ]; then \
		echo "Validating gala.yaml..."; \
		./$(BINARY_NAME) --config gala.yaml --help > /dev/null && echo "Configuration valid"; \
	else \
		echo "No gala.yaml found"; \
	fi

# Check binary size
size: build
	@echo "Binary size:"
	@ls -lh $(BINARY_NAME)
	@echo "Stripped size:"
	@strip $(BINARY_NAME) && ls -lh $(BINARY_NAME)

# Quick development cycle
dev: fmt lint test build

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  build-all       - Build for all supported platforms"
	@echo "  completions     - Generate shell completions"
	@echo "  install-completions - Install shell completions system-wide"
	@echo "  clean           - Remove build artifacts"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  benchmark       - Run benchmark tests"
	@echo "  deps            - Download dependencies"
	@echo "  tidy            - Tidy dependencies"
	@echo "  vendor          - Vendor dependencies"
	@echo "  install         - Install binary to GOPATH/bin"
	@echo "  uninstall       - Remove binary from GOPATH/bin"
	@echo "  run             - Build and run the application"
	@echo "  run-example     - Build and run with example arguments"
	@echo "  run-user        - Build and run user analysis"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  lint-fix        - Fix linting issues"
	@echo "  security        - Run security scan"
	@echo "  docs            - Generate documentation"
	@echo "  dev-setup       - Install development tools"
	@echo "  release         - Create release archives for all platforms"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run in Docker container"
	@echo "  nix-build       - Build with Nix"
	@echo "  nix-shell       - Enter Nix development shell"
	@echo "  check-updates   - Check for dependency updates"
	@echo "  validate-config - Validate configuration file"
	@echo "  size            - Show binary size"
	@echo "  dev             - Quick development cycle (fmt + lint + test + build)"
	@echo "  help            - Show this help message"
