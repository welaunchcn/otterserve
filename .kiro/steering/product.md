# Product Overview

Otter Serve Service is a lightweight, cross-platform HTTP file server written in Go. It serves static files from configurable local filesystem paths with optional basic authentication and can run both as a console application and as a system service.

## Core Features

- Cross-platform binary support (Windows and Linux)
- Configurable HTTP routing to filesystem directories
- Optional basic authentication
- System service installation/management capabilities
- YAML-based configuration
- Structured logging with configurable levels
- Docker containerization support

## Primary Use Cases

- Serving static files and documentation from local directories
- Development file server with authentication
- Production file hosting with service management
- Containerized file serving in Docker environments

## Target Platforms

- Windows (amd64) - with Windows Service support
- Linux (amd64) - with systemd service support
- Docker containers (Alpine-based)