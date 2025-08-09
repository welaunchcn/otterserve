# Otter Serve Service Build Configuration

# Application information
APP_NAME := otterserve
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT) -w -s"

# Directories
BUILD_DIR := build
DIST_DIR := dist

# Default target
.PHONY: all
all: clean test build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build for current platform
.PHONY: build
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) .

# Build for all supported platforms
.PHONY: build-all
build-all: build-windows build-linux

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

# Create distribution packages
.PHONY: dist
dist: build-all
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	
	# Windows distribution
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64
	@cp $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/
	@cp config.yaml $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/
	@cp README.md $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/
	@cp LICENSE $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/static
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/docs
	@cp static/* $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/static/ 2>/dev/null || true
	@cp docs/* $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/docs/ 2>/dev/null || true
	@cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-$(VERSION)-windows-amd64/
	
	# Linux distribution
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64
	@cp $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/$(APP_NAME)
	@cp config.yaml $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/
	@cp README.md $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/
	@cp LICENSE $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/static
	@mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/docs
	@cp static/* $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/static/ 2>/dev/null || true
	@cp docs/* $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/docs/ 2>/dev/null || true
	@cd $(DIST_DIR) && tar -czf $(APP_NAME)-$(VERSION)-linux-amd64.tar.gz $(APP_NAME)-$(VERSION)-linux-amd64/
	
	@echo "Distribution packages created in $(DIST_DIR)/"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

# Run the application in development mode
.PHONY: run
run: build
	@echo "Running application..."
	@./$(BUILD_DIR)/$(APP_NAME)

# Run with custom config
.PHONY: run-config
run-config: build
	@echo "Running application with custom config..."
	@./$(BUILD_DIR)/$(APP_NAME) -config $(CONFIG)

# Install the service (requires admin/root privileges)
.PHONY: install-service
install-service: build
	@echo "Installing service..."
	@./$(BUILD_DIR)/$(APP_NAME) -install

# Uninstall the service (requires admin/root privileges)
.PHONY: uninstall-service
uninstall-service: build
	@echo "Uninstalling service..."
	@./$(BUILD_DIR)/$(APP_NAME) -uninstall

# Development server with hot reload (requires air: go install github.com/cosmtrek/air@latest)
.PHONY: dev
dev:
	@echo "Starting development server with hot reload..."
	@air || echo "air not installed, run: go install github.com/cosmtrek/air@latest"

# Benchmark tests
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Security scan (requires gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
.PHONY: security
security:
	@echo "Running security scan..."
	@gosec ./... || echo "gosec not installed, run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all              - Clean, test, and build"
	@echo "  clean            - Clean build artifacts"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  build            - Build for current platform"
	@echo "  build-all        - Build for all supported platforms"
	@echo "  build-windows    - Build for Windows"
	@echo "  build-linux      - Build for Linux"
	@echo "  dist             - Create distribution packages"
	@echo "  deps             - Install dependencies"
	@echo "  fmt              - Format code"
	@echo "  lint             - Lint code"
	@echo "  run              - Run the application"
	@echo "  run-config       - Run with custom config (CONFIG=path)"
	@echo "  install-service  - Install as system service"
	@echo "  uninstall-service- Uninstall system service"
	@echo "  dev              - Start development server with hot reload"
	@echo "  bench            - Run benchmark tests"
	@echo "  security         - Run security scan"
	@echo "  docs             - Generate and serve documentation"
	@echo "  help             - Show this help message"