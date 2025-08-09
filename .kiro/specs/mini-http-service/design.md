# Design Document

## Overview

The mini HTTP service will be built as a Go application using the standard library's `net/http` package for HTTP handling and third-party libraries for cross-platform service management. The architecture follows a modular design with clear separation between configuration, routing, authentication, and service management concerns.

## Architecture

The application follows a layered architecture:

```
┌─────────────────────────────────────┐
│           CLI Layer                 │
│  (Service Install/Uninstall/Run)    │
├─────────────────────────────────────┤
│         Service Layer               │
│    (Cross-platform service mgmt)    │
├─────────────────────────────────────┤
│          HTTP Layer                 │
│   (Router, Middleware, Handlers)    │
├─────────────────────────────────────┤
│        Business Layer               │
│  (Auth, File Serving, Logging)      │
├─────────────────────────────────────┤
│       Configuration Layer           │
│     (Config loading/validation)     │
└─────────────────────────────────────┘
```

## Components and Interfaces

### 1. Configuration Component

**Purpose:** Manages application configuration loading, validation, and defaults.

**Key Types:**
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Auth     AuthConfig     `yaml:"auth"`
    Routes   []RouteConfig  `yaml:"routes"`
    Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
}

type AuthConfig struct {
    Enabled  bool   `yaml:"enabled"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

type RouteConfig struct {
    Path      string `yaml:"path"`
    Directory string `yaml:"directory"`
}
```

**Interface:**
```go
type ConfigManager interface {
    Load(configPath string) (*Config, error)
    Save(config *Config, configPath string) error
    Validate(config *Config) error
}
```

### 2. Service Management Component

**Purpose:** Handles cross-platform service installation, uninstallation, and lifecycle management.

**Interface:**
```go
type ServiceManager interface {
    Install(serviceName, displayName, description string) error
    Uninstall(serviceName string) error
    Start(serviceName string) error
    Stop(serviceName string) error
    Run() error
}
```

**Implementation:** Will use `github.com/kardianos/service` library for cross-platform service management.

### 3. HTTP Server Component

**Purpose:** Manages HTTP server lifecycle, routing, and middleware.

**Key Types:**
```go
type HTTPServer struct {
    config     *Config
    router     *http.ServeMux
    server     *http.Server
    middleware []Middleware
}

type Middleware func(http.Handler) http.Handler
```

**Interface:**
```go
type Server interface {
    Start() error
    Stop() error
    RegisterRoutes(routes []RouteConfig) error
}
```

### 4. Authentication Component

**Purpose:** Handles basic authentication when enabled.

**Interface:**
```go
type Authenticator interface {
    Authenticate(username, password string) bool
    Middleware() Middleware
}
```

### 5. File Server Component

**Purpose:** Serves static files from configured directories with proper MIME types and directory listing.

**Interface:**
```go
type FileServer interface {
    ServeFiles(basePath, directory string) http.Handler
    ListDirectory(directory string) ([]FileInfo, error)
}
```

### 6. Logging Component

**Purpose:** Provides structured logging with different output targets based on execution mode.

**Interface:**
```go
type Logger interface {
    Info(msg string, fields ...Field)
    Error(msg string, err error, fields ...Field)
    Debug(msg string, fields ...Field)
    SetLevel(level LogLevel)
}
```

## Data Models

### Configuration File Structure (YAML)

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
    directory: "/var/www/docs"

logging:
  level: "info"
  file: ""  # empty means stdout/stderr
```

### Service Configuration

- **Windows:** Registered as Windows Service using Service Control Manager
- **Linux:** Registered as systemd service with appropriate unit file
- **Service Name:** `mini-http-service`
- **Display Name:** `Mini HTTP Service`
- **Description:** `Lightweight HTTP file server with configurable routing`

## Error Handling

### Configuration Errors
- Invalid YAML format: Log error, use defaults, continue startup
- Missing directories in routes: Log warning, skip invalid routes
- Invalid host/port: Log error, use defaults (localhost:8080)
- Missing config file: Create default config file, log info

### Runtime Errors
- File not found: Return HTTP 404 with appropriate message
- Permission denied: Return HTTP 403 with generic message
- Authentication failure: Return HTTP 401 with WWW-Authenticate header
- Server startup failure: Log error and exit with non-zero code

### Service Management Errors
- Installation failure: Log detailed error, exit with code 1
- Uninstallation failure: Log warning, attempt cleanup, exit with code 1
- Permission denied: Log error with suggestion to run as administrator/root

## Testing Strategy

### Unit Tests
- Configuration loading and validation
- Authentication logic
- File serving functionality
- Route matching and resolution
- Service management operations (mocked)

### Integration Tests
- HTTP server startup and shutdown
- End-to-end request handling with authentication
- File serving with various MIME types
- Directory listing functionality
- Configuration file changes

### Platform Tests
- Service installation/uninstallation on Windows
- Service installation/uninstallation on Linux
- Cross-platform binary compatibility
- File path handling differences between platforms

### Test Structure
```
tests/
├── unit/
│   ├── config_test.go
│   ├── auth_test.go
│   ├── fileserver_test.go
│   └── service_test.go
├── integration/
│   ├── server_test.go
│   └── routes_test.go
└── platform/
    ├── windows_test.go
    └── linux_test.go
```

### Build and Deployment
- Use Go build tags for platform-specific code
- Cross-compilation for Windows and Linux targets
- Automated testing in CI/CD pipeline for both platforms
- Binary packaging with default configuration files