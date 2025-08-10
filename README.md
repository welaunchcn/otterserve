# Otter Serve Service

A lightweight HTTP service in Go that can run as a cross-platform binary on Windows and Linux. The service provides configurable routing to local file system paths with optional basic authentication, and includes service management capabilities.

## Features

- Cross-platform binary (Windows and Linux)
- Configurable routing to file system paths
- Optional basic authentication
- Service management (install/uninstall)
- YAML configuration
- Structured logging

## Usage

### Basic Usage
```bash
# Run in console mode
./otterserve

# Install as system service
./otterserve -install

# Uninstall system service
./otterserve -uninstall

# Show version
./otterserve -version

# Show help
./otterserve -help
```

### Configuration

The service reads configuration from `config.yaml`. If the file doesn't exist, default values are used.

Example configuration:
```yaml
server:
  host: "localhost"
  port: 8080

auth:
  enabled: false
  username: ""
  password: ""

routes:
  - path: "/static"
    directory: "./static"
  - path: "/docs"
    directory: "./docs"

logging:
  level: "info"
  file: ""  # empty means stdout/stderr
```

## Building

### Using Make (Linux/macOS)

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Create distribution packages
make dist

# Run tests
make test

# Clean build artifacts
make clean
```

### Using PowerShell (Windows)

```powershell
# Build for current platform
.\scripts\build.ps1 -Target build

# Build for all platforms
.\scripts\build.ps1 -Target build-all

# Create distribution packages
.\scripts\build.ps1 -Target dist

# Run tests
.\scripts\build.ps1 -Target test
```

### Using Bash (Linux/macOS/WSL)

```bash
# Build for current platform
./scripts/build.sh build

# Build for all platforms
./scripts/build.sh build-all

# Create distribution packages
./scripts/build.sh dist

# Run tests
./scripts/build.sh test
```

### Manual Build

```bash
# Build for current platform
go build -o otterserve ./cmd/otterserve

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o otterserve.exe ./cmd/otterserve

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o otterserve ./cmd/otterserve
```

### Docker Build

```bash
# Build Docker image
docker build -t otterserve .

# Run in Docker
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml otterserve
```

## Development

The project structure follows Go best practices:

```
├── cmd/otterserve/           # Application entry point and tests
├── config.yaml               # Default configuration
├── internal/
│   ├── config/               # Configuration management
│   ├── auth/                 # Authentication components
│   ├── server/               # HTTP server components
│   ├── fileserver/           # File serving components
│   ├── service/              # Service management
│   └── logger/               # Logging components
├── scripts/                  # Build scripts
├── static/                   # Example static files
└── docs/                     # Documentation and test summaries
```
