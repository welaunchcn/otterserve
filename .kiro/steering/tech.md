# Technology Stack

## Core Technologies

- **Language**: Go 1.21+
- **Configuration**: YAML (gopkg.in/yaml.v3)
- **Service Management**: github.com/kardianos/service
- **Container**: Docker with Alpine Linux base

## Build System

The project supports multiple build approaches:

### Make (Linux/macOS)
```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make test           # Run tests
make dist           # Create distribution packages
make clean          # Clean build artifacts
```

### PowerShell (Windows)
```powershell
.\build.ps1 -Target build        # Build for current platform
.\build.ps1 -Target build-all    # Build for all platforms
.\build.ps1 -Target test         # Run tests
.\build.ps1 -Target dist         # Create distribution packages
```

### Bash Script (Unix-like)
```bash
./build.sh build        # Build for current platform
./build.sh build-all    # Build for all platforms
./build.sh test         # Run tests
./build.sh dist         # Create distribution packages
```

### Docker
```bash
docker build -t otterserve .
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml otterserve
```

## Common Commands

- **Development**: `make run` or `.\build.ps1 -Target run`
- **Testing**: `go test -v ./...`
- **Formatting**: `go fmt ./...`
- **Service Install**: `./otterserve -install`
- **Service Uninstall**: `./otterserve -uninstall`

## Build Flags

The build system uses ldflags to embed version information:
- Version, build time, and git commit are embedded at compile time
- Binaries are stripped (`-w -s`) for smaller size
- Cross-compilation supported for Windows and Linux amd64