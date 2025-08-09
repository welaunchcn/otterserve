package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"otterserve/internal/auth"
	"otterserve/internal/config"
	"otterserve/internal/fileserver"
	"otterserve/internal/logger"
)

// Server interface defines HTTP server operations
type Server interface {
	Start() error
	Stop(ctx context.Context) error
	RegisterRoutes(routes []config.RouteConfig) error
	GetAddr() string
}

// HTTPServer implements the Server interface
type HTTPServer struct {
	config       *config.Config
	server       *http.Server
	mux          *http.ServeMux
	logger       logger.Logger
	authenticator auth.Authenticator
	fileServer   fileserver.FileServer
	actualAddr   string
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(cfg *config.Config, log logger.Logger, authenticator auth.Authenticator, fileServer fileserver.FileServer) Server {
	mux := http.NewServeMux()
	
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &HTTPServer{
		config:        cfg,
		server:        server,
		mux:           mux,
		logger:        log,
		authenticator: authenticator,
		fileServer:    fileServer,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Register routes from configuration
	if err := s.RegisterRoutes(s.config.Routes); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
	}

	s.logger.Info("Starting HTTP server", logger.Fields{
		"address": s.server.Addr,
		"routes":  len(s.config.Routes),
		"auth_enabled": s.authenticator.IsEnabled(),
	})

	// Create listener to get actual address
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Store the actual address
	s.actualAddr = listener.Addr().String()

	// Start server in a goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server failed to start", logger.Fields{
				"error": err.Error(),
			})
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	s.logger.Info("HTTP server started successfully", logger.Fields{
		"address": s.actualAddr,
	})

	return nil
}

// Stop gracefully stops the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to gracefully shutdown server", logger.Fields{
			"error": err.Error(),
		})
		return err
	}
	
	s.logger.Info("HTTP server stopped")
	return nil
}

// RegisterRoutes registers file serving routes from configuration
func (s *HTTPServer) RegisterRoutes(routes []config.RouteConfig) error {
	if len(routes) == 0 {
		return fmt.Errorf("no routes configured")
	}

	for _, route := range routes {
		if err := s.registerRoute(route); err != nil {
			return fmt.Errorf("failed to register route %s: %w", route.Path, err)
		}
	}

	// Register 404 handler for unmatched routes
	s.mux.HandleFunc("/", s.notFoundHandler)

	return nil
}

// registerRoute registers a single route with middleware chain
func (s *HTTPServer) registerRoute(route config.RouteConfig) error {
	// Validate route configuration
	if route.Path == "" {
		return fmt.Errorf("route path cannot be empty")
	}
	if route.Directory == "" {
		return fmt.Errorf("route directory cannot be empty")
	}

	// Ensure path starts with / and ends with /
	path := route.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	s.logger.Info("Registering route", logger.Fields{
		"path":      path,
		"directory": route.Directory,
	})

	// Create file serving handler
	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.fileServer.ServeFiles(w, r, path, route.Directory)
	})

	// Apply middleware chain: logging -> authentication -> file serving
	handler := s.loggingMiddleware(s.authenticator.Middleware(fileHandler))

	// Register the handler
	s.mux.Handle(path, handler)

	return nil
}

// loggingMiddleware logs HTTP requests
func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create request-specific logger
		requestLogger := s.logger.(*logger.DefaultLogger).RequestLogger(
			generateRequestID(),
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start
		requestLogger.Info("Request started", logger.Fields{
			"user_agent": r.UserAgent(),
		})

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request completion
		duration := time.Since(start)
		requestLogger.Info("Request completed", logger.Fields{
			"status_code": wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"bytes":       wrapped.bytesWritten,
		})
	})
}

// notFoundHandler handles requests that don't match any registered routes
func (s *HTTPServer) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	// Check if any route prefix matches
	for _, route := range s.config.Routes {
		routePath := route.Path
		if !strings.HasPrefix(routePath, "/") {
			routePath = "/" + routePath
		}
		if !strings.HasSuffix(routePath, "/") {
			routePath = routePath + "/"
		}
		
		if strings.HasPrefix(r.URL.Path, routePath) {
			// This should have been handled by the route handler
			// If we're here, it means the file wasn't found
			return
		}
	}

	// No matching route found
	s.logger.Info("Route not found", logger.Fields{
		"path":        r.URL.Path,
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 Not Found\n\nThe requested path '%s' was not found on this server.\n", r.URL.Path)
}

// GetAddr returns the server address
func (s *HTTPServer) GetAddr() string {
	if s.actualAddr != "" {
		return s.actualAddr
	}
	return s.server.Addr
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written
func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.bytesWritten += int64(n)
	return n, err
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// LifecycleManager manages server lifecycle including graceful shutdown
type LifecycleManager struct {
	server Server
	logger logger.Logger
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(server Server, logger logger.Logger) *LifecycleManager {
	return &LifecycleManager{
		server: server,
		logger: logger,
	}
}

// Run starts the server and handles graceful shutdown
func (lm *LifecycleManager) Run(ctx context.Context) error {
	// Start the server
	if err := lm.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	lm.logger.Info("Server lifecycle manager started")

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	lm.logger.Info("Shutdown signal received, starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := lm.server.Stop(shutdownCtx); err != nil {
		lm.logger.Error("Failed to gracefully shutdown server", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	lm.logger.Info("Server shutdown completed")
	return nil
}

// GetServerAddr returns the server address
func (lm *LifecycleManager) GetServerAddr() string {
	return lm.server.GetAddr()
}