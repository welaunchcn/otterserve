# Project Structure

## Directory Layout

```
├── main.go                    # Entry point with CLI parsing and service management
├── config.yaml               # Default YAML configuration file
├── go.mod / go.sum           # Go module dependencies
├── Dockerfile                # Multi-stage Docker build configuration
├── Makefile                  # Unix build automation
├── build.ps1 / build.sh      # Cross-platform build scripts
├── README.md                 # Project documentation
├── LICENSE                   # License file
├── static/                   # Example static files directory
├── docs/                     # Example documentation directory
└── internal/                 # Private application packages
    ├── auth/                 # Authentication components
    ├── config/               # Configuration management
    ├── fileserver/           # File serving logic
    ├── logger/               # Structured logging
    ├── server/               # HTTP server lifecycle management
    └── service/              # System service management
```

## Architecture Patterns

### Package Organization
- **internal/**: All private packages following Go best practices
- Each package has a single responsibility (auth, config, logger, etc.)
- Interfaces defined for testability and dependency injection
- Test files co-located with implementation (`*_test.go`)

### Key Interfaces
- `ConfigManager`: Configuration loading and validation
- `Logger`: Structured logging with levels and fields
- `ServiceManager`: System service installation and management
- `AuthProvider`: Basic authentication handling

### Configuration Management
- YAML-based configuration with struct tags
- Default configuration creation if file doesn't exist
- Validation with detailed error messages
- Environment-specific overrides supported

### Error Handling
- Structured error messages with context
- Graceful degradation where possible
- Detailed logging for troubleshooting
- Clean shutdown handling with signal management

### Testing Strategy
- Unit tests for all packages
- Integration tests for full application flow
- Test coverage reporting available via build scripts
- Mock interfaces for external dependencies