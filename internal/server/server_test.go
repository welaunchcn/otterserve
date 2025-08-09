package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mini-http-service/internal/auth"
	"mini-http-service/internal/config"
	"mini-http-service/internal/fileserver"
	"mini-http-service/internal/logger"
)

func TestNewHTTPServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
	}
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer)
	if server == nil {
		t.Fatal("Expected server to be created")
	}

	httpServer, ok := server.(*HTTPServer)
	if !ok {
		t.Fatal("Expected HTTPServer type")
	}

	if httpServer.GetAddr() != "localhost:1124" {
		t.Errorf("Expected address 'localhost:1124', got '%s'", httpServer.GetAddr())
	}
}

func TestHTTPServer_RegisterRoutes(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
	}
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	// Create temporary directories for testing
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	docsDir := filepath.Join(tempDir, "docs")
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(docsDir, 0755)

	routes := []config.RouteConfig{
		{Path: "/static", Directory: staticDir},
		{Path: "/docs", Directory: docsDir},
	}

	err := server.RegisterRoutes(routes)
	if err != nil {
		t.Fatalf("Failed to register routes: %v", err)
	}
}

func TestHTTPServer_RegisterRoutes_EmptyRoutes(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
	}
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	err := server.RegisterRoutes([]config.RouteConfig{})
	if err == nil {
		t.Error("Expected error for empty routes")
	}
}

func TestHTTPServer_RegisterRoutes_InvalidRoute(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
	}
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	routes := []config.RouteConfig{
		{Path: "", Directory: "/tmp"}, // Empty path
	}

	err := server.RegisterRoutes(routes)
	if err == nil {
		t.Error("Expected error for invalid route")
	}
}

func TestHTTPServer_NotFoundHandler(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
		Routes: []config.RouteConfig{
			{Path: "/static", Directory: "/tmp"},
		},
	}
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()

	server.notFoundHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "404 Not Found") {
		t.Error("Expected 404 message in response body")
	}
	if !strings.Contains(body, "/nonexistent") {
		t.Error("Expected requested path in response body")
	}
}

func TestHTTPServer_LoggingMiddleware(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
	}
	
	// Use a buffer to capture log output
	var logBuffer strings.Builder
	log := logger.NewLogger(logger.InfoLevel, &logBuffer)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	wrappedHandler := server.loggingMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	logOutput := logBuffer.String()
	
	// Check that request was logged
	if !strings.Contains(logOutput, "Request started") {
		t.Error("Expected 'Request started' in log output")
	}
	if !strings.Contains(logOutput, "Request completed") {
		t.Error("Expected 'Request completed' in log output")
	}
	if !strings.Contains(logOutput, "test-agent") {
		t.Error("Expected user agent in log output")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rw.statusCode)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
	if rw.bytesWritten != int64(len(data)) {
		t.Errorf("Expected %d bytes tracked, got %d", len(data), rw.bytesWritten)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Expected different request IDs")
	}

	if !strings.HasPrefix(id1, "req-") {
		t.Errorf("Expected request ID to start with 'req-', got '%s'", id1)
	}
}

func TestHTTPServer_Integration(t *testing.T) {
	// Create temporary directory with test file
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	os.MkdirAll(staticDir, 0755)
	
	testFile := filepath.Join(staticDir, "test.txt")
	testContent := "Hello, World!"
	os.WriteFile(testFile, []byte(testContent), 0644)

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
		Routes: []config.RouteConfig{
			{Path: "/static", Directory: staticDir},
		},
	}
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer).(*HTTPServer)

	// Register routes
	err := server.RegisterRoutes(cfg.Routes)
	if err != nil {
		t.Fatalf("Failed to register routes: %v", err)
	}

	// Test file serving
	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, rr.Body.String())
	}
}

func TestHTTPServer_StartStop(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0}, // Use port 0 for testing
		Routes: []config.RouteConfig{
			{Path: "/test", Directory: t.TempDir()},
		},
	}
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer)

	// Start server
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

func TestLifecycleManager_NewLifecycleManager(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 1124},
		Routes: []config.RouteConfig{
			{Path: "/test", Directory: t.TempDir()},
		},
	}
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()
	server := NewHTTPServer(cfg, log, authenticator, fileServer)

	lm := NewLifecycleManager(server, log)
	if lm == nil {
		t.Fatal("Expected lifecycle manager to be created")
	}

	if lm.GetServerAddr() != server.GetAddr() {
		t.Errorf("Expected server address '%s', got '%s'", server.GetAddr(), lm.GetServerAddr())
	}
}

func TestLifecycleManager_Run(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0}, // Use port 0 for testing
		Routes: []config.RouteConfig{
			{Path: "/test", Directory: t.TempDir()},
		},
	}
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()
	server := NewHTTPServer(cfg, log, authenticator, fileServer)

	lm := NewLifecycleManager(server, log)

	// Create context that will be cancelled to trigger shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Run lifecycle manager in goroutine
	done := make(chan error, 1)
	go func() {
		done <- lm.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Trigger shutdown
	cancel()

	// Wait for shutdown to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Lifecycle manager returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Lifecycle manager did not shutdown within timeout")
	}
}

func TestHTTPServer_StartWithAuthEnabled(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
		Routes: []config.RouteConfig{
			{Path: "/test", Directory: t.TempDir()},
		},
	}
	
	var logBuffer strings.Builder
	log := logger.NewLogger(logger.InfoLevel, &logBuffer)
	authenticator := auth.NewBasicAuthenticator(true, "admin", "secret")
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer)

	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Check that auth enabled was logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "auth_enabled=true") {
		t.Error("Expected auth_enabled=true in log output")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)
}

func TestHTTPServer_GracefulShutdownTimeout(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
		Routes: []config.RouteConfig{
			{Path: "/test", Directory: t.TempDir()},
		},
	}
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	server := NewHTTPServer(cfg, log, authenticator, fileServer)

	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Create a very short timeout context to test timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should complete quickly since the server isn't handling any requests
	err = server.Stop(ctx)
	// We don't check for error here as the timeout might or might not occur
	// depending on timing, but the important thing is that it doesn't hang
}