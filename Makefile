# Makefile for AI Trading Bot

# Variables
APP_NAME := trading-bot
VERSION := 1.0.0
BUILD_DIR := build
DIST_DIR := dist
CONFIG_FILE := config.json
MAIN_PACKAGE := cmd/main.go

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Build flags
LDFLAGS := -ldflags "-X main.AppVersion=$(VERSION) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Platform settings
PLATFORMS := linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

# Default target
.PHONY: all
all: clean deps test build

# Help target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependencies
.PHONY: deps
deps: ## Install dependencies
	$(GOMOD) download
	$(GOMOD) tidy

# Build targets
.PHONY: build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PACKAGE)

.PHONY: build-release
build-release: ## Build release binaries for all platforms
	@echo "Building release binaries..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		echo "Building for $(platform)..."; \
		GOOS=$(word 1,$(subst /, ,$(platform))) GOARCH=$(word 2,$(subst /, ,$(platform))) \
			$(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-$(platform) $(MAIN_PACKAGE); \
	)

.PHONY: build-linux
build-linux: ## Build for Linux
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux $(MAIN_PACKAGE)

.PHONY: build-windows
build-windows: ## Build for Windows
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME).exe $(MAIN_PACKAGE)

.PHONY: build-darwin
build-darwin: ## Build for macOS
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin $(MAIN_PACKAGE)

# Testing
.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GOTEST) -v -race ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Linting and formatting
.PHONY: fmt
fmt: ## Format Go code
	$(GOFMT) -s -w .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt' to fix." && exit 1)

.PHONY: lint
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2)
	golangci-lint run

.PHONY: vet
vet: ## Run go vet
	$(GOCMD) vet ./...

# Quality checks
.PHONY: check
check: fmt-check vet lint test ## Run all quality checks

# Configuration
.PHONY: config
config: ## Create default configuration file
	@if [ ! -f $(CONFIG_FILE) ]; then \
		echo "Creating default configuration file: $(CONFIG_FILE)"; \
		echo '{"app":{"name":"AI Trading Bot","version":"$(VERSION)","environment":"development","debug":true},"trading":{"initial_balance":10000,"supported_symbols":["BTCUSDT"],"default_symbol":"BTCUSDT","execution_type":"simulation"},"strategy":{"grid":{"min_grid_levels":10,"max_grid_levels":30},"breakout":{"confirmation_candles":3},"false_breakout":{"price_reversion_threshold":0.005},"stability":{"analysis_window":10}},"risk":{"max_portfolio_risk":0.05,"max_position_risk":0.02},"stream":{"provider_type":"simulation"},"logging":{"level":"info","format":"json","output":"both"}}' > $(CONFIG_FILE); \
	else \
		echo "Configuration file already exists: $(CONFIG_FILE)"; \
	fi

# Running the application
.PHONY: run
run: build ## Build and run the application
	./$(BUILD_DIR)/$(APP_NAME) -config $(CONFIG_FILE)

.PHONY: run-debug
run-debug: build ## Build and run with debug mode
	./$(BUILD_DIR)/$(APP_NAME) -config $(CONFIG_FILE) -debug

.PHONY: run-test
run-test: ## Run test mode with simulated data
	@echo "Running test mode..."
	./$(BUILD_DIR)/$(APP_NAME) -config $(CONFIG_FILE) -debug

# Development tools
.PHONY: dev
dev: ## Set up development environment
	@echo "Setting up development environment..."
	$(MAKE) deps
	$(MAKE) config
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi
	@echo "Development environment ready!"

.PHONY: tools
tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: watch
watch: ## Run development server with hot reload (requires air)
	@if ! command -v air &> /dev/null; then \
		echo "Installing air for hot reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	air -c .air.toml

# Documentation
.PHONY: docs
docs: ## Generate documentation
	@if ! command -v swag &> /dev/null; then \
		echo "Installing swag for documentation..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	swag init -g cmd/main.go -o docs

.PHONY: docs-serve
docs-serve: ## Serve documentation
	@echo "Documentation available at: http://localhost:6060/pkg/"
	godoc -http=:6060


# Database
.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	# Add migration commands here

.PHONY: db-reset
db-reset: ## Reset database
	@echo "Resetting database..."
	# Add database reset commands here

# Cleanup
.PHONY: clean
clean: ## Clean build artifacts
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html

.PHONY: clean-all
clean-all: clean ## Clean all generated files
	@rm -rf logs/
	@rm -rf data/
	@rm -f *.log
	@rm -f *.db

# Installation
.PHONY: install
install: build ## Install the application
	@echo "Installing $(APP_NAME) to $(GOPATH)/bin..."
	cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/

.PHONY: install-local
install-local: build ## Install locally to /usr/local/bin
	@echo "Installing $(APP_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --rm $(APP_NAME):$(VERSION)

# Release
.PHONY: release
release: clean test check build-release ## Prepare release
	@echo "Release prepared in $(DIST_DIR)/"

# Version
.PHONY: version
version: ## Show version information
	@echo "$(APP_NAME) version $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Build time: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Security
.PHONY: security
security: ## Run security checks
	@echo "Running security checks..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

.PHONY: vuln-check
vuln-check: ## Check for known vulnerabilities
	@echo "Checking for vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Performance
.PHONY: profile
profile: ## Generate CPU profile
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -cpuprofile=$(BUILD_DIR)/cpu.prof -memprofile=$(BUILD_DIR)/mem.prof -bench=. ./...

.PHONY: profile-cpu
profile-cpu: profile ## Analyze CPU profile
	@echo "Analyzing CPU profile..."
	$(GOCMD) tool pprof $(BUILD_DIR)/cpu.prof

.PHONY: profile-mem
profile-mem: profile ## Analyze memory profile
	@echo "Analyzing memory profile..."
	$(GOCMD) tool pprof $(BUILD_DIR)/mem.prof

# CI/CD helpers
.PHONY: ci
ci: deps fmt-check vet test security ## Run CI checks
	@echo "All CI checks passed!"

.PHONY: pre-commit
pre-commit: fmt-check vet test ## Run pre-commit checks
	@echo "Pre-commit checks passed!"

# Project setup
.PHONY: init
init: ## Initialize a new project
	@echo "Initializing AI Trading Bot project..."
	$(MAKE) dev
	$(MAKE) config
	@echo "Project initialized successfully!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Edit $(CONFIG_FILE) to configure your trading parameters"
	@echo "2. Run 'make run' to start the bot"
	@echo "3. Run 'make test' to run tests"
	@echo "4. Run 'make docs' to generate documentation"

# Quick commands for development
.PHONY: q
q: run ## Quick run (alias for run)

.PHONY: t
t: test ## Quick test (alias for test)

.PHONY: b
b: build ## Quick build (alias for build)

.PHONY: c
c: clean ## Quick clean (alias for clean)

# Default help message if no target is specified
.DEFAULT:
	@echo "Unknown target. Run 'make help' to see available targets."