# Multi-stage build for minimal production image
FROM golang:1.21-alpine AS builder

# Install git for version info
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -w -s" \
    -o otterserve ./cmd/otterserve

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/otterserve .

# Copy default configuration
COPY config.yaml .

# Create directories for file serving
RUN mkdir -p static docs && \
    chown -R appuser:appgroup /app

# Create sample files
RUN echo '<!DOCTYPE html><html><head><title>Otter Serve Service</title></head><body><h1>Welcome to Otter Serve Service</h1><p>This service is running in a Docker container.</p></body></html>' > static/index.html && \
    echo 'Otter Serve Service Documentation\n\nThis is a lightweight HTTP file server running in Docker.' > docs/readme.txt && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/static/ || exit 1

# Run the application
CMD ["./otterserve"]
