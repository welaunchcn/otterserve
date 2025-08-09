#!/bin/bash

# Build script for Unix-like systems (Linux, macOS)

set -e

APP_NAME="otterserve"
VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS="-ldflags \"-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT -w -s\""
BUILD_DIR="build"
DIST_DIR="dist"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

function log_info() {
    echo -e "${GREEN}$1${NC}"
}

function log_warn() {
    echo -e "${YELLOW}$1${NC}"
}

function log_error() {
    echo -e "${RED}$1${NC}"
}

function clean() {
    log_info "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR" "$DIST_DIR"
    go clean
}

function test_app() {
    log_info "Running tests..."
    go test -v ./...
}

function test_coverage() {
    log_info "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    log_warn "Coverage report generated: coverage.html"
}

function build() {
    log_info "Building for current platform..."
    mkdir -p "$BUILD_DIR"
    eval "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME ."
}

function build_windows() {
    log_info "Building for Windows..."
    mkdir -p "$BUILD_DIR"
    GOOS=windows GOARCH=amd64 eval "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME-windows-amd64.exe ."
}

function build_linux() {
    log_info "Building for Linux..."
    mkdir -p "$BUILD_DIR"
    GOOS=linux GOARCH=amd64 eval "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME-linux-amd64 ."
}

function build_all() {
    build_windows
    build_linux
}

function create_dist() {
    log_info "Creating distribution packages..."
    build_all
    
    mkdir -p "$DIST_DIR"
    
    # Windows distribution
    WIN_DIR="$DIST_DIR/$APP_NAME-$VERSION-windows-amd64"
    mkdir -p "$WIN_DIR"
    cp "$BUILD_DIR/$APP_NAME-windows-amd64.exe" "$WIN_DIR/"
    cp config.yaml "$WIN_DIR/" 2>/dev/null || true
    cp README.md "$WIN_DIR/" 2>/dev/null || true
    cp LICENSE "$WIN_DIR/" 2>/dev/null || true
    mkdir -p "$WIN_DIR/static" "$WIN_DIR/docs"
    cp static/* "$WIN_DIR/static/" 2>/dev/null || true
    cp docs/* "$WIN_DIR/docs/" 2>/dev/null || true
    
    # Create zip file
    (cd "$DIST_DIR" && zip -r "$APP_NAME-$VERSION-windows-amd64.zip" "$APP_NAME-$VERSION-windows-amd64/")
    
    # Linux distribution
    LINUX_DIR="$DIST_DIR/$APP_NAME-$VERSION-linux-amd64"
    mkdir -p "$LINUX_DIR"
    cp "$BUILD_DIR/$APP_NAME-linux-amd64" "$LINUX_DIR/$APP_NAME"
    cp config.yaml "$LINUX_DIR/" 2>/dev/null || true
    cp README.md "$LINUX_DIR/" 2>/dev/null || true
    cp LICENSE "$LINUX_DIR/" 2>/dev/null || true
    mkdir -p "$LINUX_DIR/static" "$LINUX_DIR/docs"
    cp static/* "$LINUX_DIR/static/" 2>/dev/null || true
    cp docs/* "$LINUX_DIR/docs/" 2>/dev/null || true
    
    # Create tar.gz file
    (cd "$DIST_DIR" && tar -czf "$APP_NAME-$VERSION-linux-amd64.tar.gz" "$APP_NAME-$VERSION-linux-amd64/")
    
    log_warn "Distribution packages created in $DIST_DIR/"
}

function install_deps() {
    log_info "Installing dependencies..."
    go mod download
    go mod tidy
}

function format_code() {
    log_info "Formatting code..."
    go fmt ./...
}

function lint_code() {
    log_info "Linting code..."
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
    else
        log_warn "golangci-lint not installed, skipping..."
    fi
}

function run_app() {
    log_info "Running application..."
    build
    "./$BUILD_DIR/$APP_NAME"
}

function run_with_config() {
    if [ -z "$CONFIG" ]; then
        log_error "CONFIG environment variable not set"
        exit 1
    fi
    log_info "Running application with config: $CONFIG"
    build
    "./$BUILD_DIR/$APP_NAME" -config "$CONFIG"
}

function install_service() {
    log_info "Installing service..."
    build
    "./$BUILD_DIR/$APP_NAME" -install
}

function uninstall_service() {
    log_info "Uninstalling service..."
    build
    "./$BUILD_DIR/$APP_NAME" -uninstall
}

function dev_server() {
    log_info "Starting development server with hot reload..."
    if command -v air &> /dev/null; then
        air
    else
        log_warn "air not installed, run: go install github.com/cosmtrek/air@latest"
    fi
}

function benchmark() {
    log_info "Running benchmarks..."
    go test -bench=. -benchmem ./...
}

function security_scan() {
    log_info "Running security scan..."
    if command -v gosec &> /dev/null; then
        gosec ./...
    else
        log_warn "gosec not installed, run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
    fi
}

function show_help() {
    echo -e "${CYAN}Available targets:${NC}"
    echo "  all              - Clean, test, and build"
    echo "  clean            - Clean build artifacts"
    echo "  test             - Run tests"
    echo "  test-coverage    - Run tests with coverage report"
    echo "  build            - Build for current platform"
    echo "  build-all        - Build for all supported platforms"
    echo "  build-windows    - Build for Windows"
    echo "  build-linux      - Build for Linux"
    echo "  dist             - Create distribution packages"
    echo "  deps             - Install dependencies"
    echo "  fmt              - Format code"
    echo "  lint             - Lint code"
    echo "  run              - Run the application"
    echo "  run-config       - Run with custom config (CONFIG=path)"
    echo "  install-service  - Install as system service"
    echo "  uninstall-service- Uninstall system service"
    echo "  dev              - Start development server with hot reload"
    echo "  bench            - Run benchmark tests"
    echo "  security         - Run security scan"
    echo "  help             - Show this help message"
    echo ""
    echo "Usage: ./build.sh <target>"
    echo "       CONFIG=path ./build.sh run-config"
}

# Main execution
case "${1:-help}" in
    "all")
        clean
        test_app
        build
        ;;
    "clean")
        clean
        ;;
    "test")
        test_app
        ;;
    "test-coverage")
        test_coverage
        ;;
    "build")
        build
        ;;
    "build-windows")
        build_windows
        ;;
    "build-linux")
        build_linux
        ;;
    "build-all")
        build_all
        ;;
    "dist")
        create_dist
        ;;
    "deps")
        install_deps
        ;;
    "fmt")
        format_code
        ;;
    "lint")
        lint_code
        ;;
    "run")
        run_app
        ;;
    "run-config")
        run_with_config
        ;;
    "install-service")
        install_service
        ;;
    "uninstall-service")
        uninstall_service
        ;;
    "dev")
        dev_server
        ;;
    "bench")
        benchmark
        ;;
    "security")
        security_scan
        ;;
    "help")
        show_help
        ;;
    *)
        log_error "Unknown target: $1"
        show_help
        exit 1
        ;;
esac