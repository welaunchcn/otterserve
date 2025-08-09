package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mini-http-service/internal/auth"
	"mini-http-service/internal/config"
	"mini-http-service/internal/fileserver"
	"mini-http-service/internal/logger"
	"mini-http-service/internal/server"
	"mini-http-service/internal/service"
)

// TestComprehensiveScenarios tests various real-world scenarios
func TestComprehensiveScenarios(t *testing.T) {
	// Create comprehensive test environment
	tempDir := t.TempDir()
	
	// Create complex directory structure
	dirs := []string{
		"static/css",
		"static/js",
		"static/images",
		"docs/api",
		"docs/guides",
		"files/uploads",
		"files/downloads",
	}
	
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(tempDir, dir), 0755)
	}
	
	// Create various file types
	files := map[string]string{
		"static/index.html":           `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Home</h1></body></html>`,
		"static/css/style.css":        `body { font-family: Arial, sans-serif; }`,
		"static/js/app.js":            `console.log('Hello, World!');`,
		"static/images/logo.png":      "fake png data",
		"docs/readme.md":              "# Documentation\n\nThis is the main documentation.",
		"docs/api/endpoints.json":     `{"endpoints": ["/api/v1/users", "/api/v1/posts"]}`,
		"docs/guides/quickstart.txt":  "Quick Start Guide\n\n1. Install\n2. Configure\n3. Run",
		"files/uploads/data.csv":      "name,age,city\nJohn,30,NYC\nJane,25,LA",
		"files/downloads/manual.pdf":  "fake pdf content",
	}
	
	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		os.WriteFile(fullPath, []byte(content), 0644)
	}
	
	// Create configuration with multiple routes
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
		Auth: config.AuthConfig{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		Routes: []config.RouteConfig{
			{Path: "/static", Directory: filepath.Join(tempDir, "static")},
			{Path: "/docs", Directory: filepath.Join(tempDir, "docs")},
			{Path: "/files", Directory: filepath.Join(tempDir, "files")},
		},
		Logging: config.LoggingConfig{
			Level: "info",
			File:  "",
		},
	}
	
	// Start server
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewBasicAuthenticator(cfg.Auth.Enabled, cfg.Auth.Username, cfg.Auth.Password)
	fileServer := fileserver.NewFileServer()
	httpServer := server.NewHTTPServer(cfg, log, authenticator, fileServer)
	lifecycleManager := server.NewLifecycleManager(httpServer, log)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- lifecycleManager.Run(ctx)
	}()
	
	time.Sleep(200 * time.Millisecond)
	
	baseURL := fmt.Sprintf("http://%s", httpServer.GetAddr())
	
	// Comprehensive test scenarios
	scenarios := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		checkContent   string
	}{
		// Static files with different types
		{"HTML file", "/static/index.html", 200, "text/html", "<h1>Home</h1>"},
		{"CSS file", "/static/css/style.css", 200, "text/css", "font-family"},
		{"JavaScript file", "/static/js/app.js", 200, "text/javascript", "console.log"},
		{"PNG image", "/static/images/logo.png", 200, "image/png", "fake png"},
		
		// Documentation files (Windows may have different MIME types)
		{"Markdown file", "/docs/readme.md", 200, "", "Documentation"},
		{"JSON file", "/docs/api/endpoints.json", 200, "application/json", "endpoints"},
		{"Text file", "/docs/guides/quickstart.txt", 200, "text/plain", "Quick Start"},
		
		// File downloads (Windows may detect CSV as Excel)
		{"CSV file", "/files/uploads/data.csv", 200, "", "name,age,city"},
		{"PDF file", "/files/downloads/manual.pdf", 200, "application/pdf", "fake pdf"},
		
		// Directory listings (static has index.html so serves that instead of listing)
		{"Static directory", "/static/", 200, "text/html", "Home"},
		{"Docs directory", "/docs/", 200, "text/html", "api/"},
		{"Files directory", "/files/", 200, "text/html", "uploads/"},
		{"Subdirectory", "/static/css/", 200, "text/html", "style.css"},
		
		// Error cases
		{"Nonexistent file", "/static/nonexistent.txt", 404, "", ""},
		{"Nonexistent route", "/invalid/path", 404, "", "404 Not Found"},
	}
	
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + scenario.path)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != scenario.expectedStatus {
				t.Errorf("Expected status %d, got %d", scenario.expectedStatus, resp.StatusCode)
			}
			
			if scenario.expectedType != "" {
				contentType := resp.Header.Get("Content-Type")
				if !strings.Contains(contentType, scenario.expectedType) {
					t.Errorf("Expected content type to contain '%s', got '%s'", scenario.expectedType, contentType)
				}
			}
			
			if scenario.checkContent != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				
				if !strings.Contains(string(body), scenario.checkContent) {
					t.Errorf("Expected body to contain '%s', got '%s'", scenario.checkContent, string(body))
				}
			}
		})
	}
	
	cancel()
	
	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("Server returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

// TestServiceInstallationUninstallation tests service management
func TestServiceInstallationUninstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping service installation test in short mode")
	}
	
	// Create temporary config
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "service-test-config.yaml")
	staticDir := filepath.Join(tempDir, "static")
	
	os.MkdirAll(staticDir, 0755)
	os.WriteFile(filepath.Join(staticDir, "test.txt"), []byte("service test"), 0644)
	
	configContent := fmt.Sprintf(`server:
  host: "localhost"
  port: 1124
auth:
  enabled: false
routes:
  - path: "/static"
    directory: "%s"
logging:
  level: "info"
  file: ""
`, staticDir)
	
	os.WriteFile(configFile, []byte(configContent), 0644)
	
	log := logger.NewLogger(logger.InfoLevel, nil)
	
	// Test service manager creation
	_, err := service.NewServiceManager(
		"test-mini-http-service",
		"Test Mini HTTP Service",
		"Test service for unit testing",
		configFile,
		log,
	)
	if err != nil {
		t.Fatalf("Failed to create service manager: %v", err)
	}
	
	// Note: We don't actually install/uninstall the service in tests
	// as it requires admin privileges and could interfere with the system
	// Instead, we verify that the service manager was created successfully
	// and that the configuration is valid
	
	t.Log("Service manager created successfully")
	t.Log("Service installation/uninstallation would require admin privileges")
}

// TestConfigurationEdgeCases tests various configuration edge cases
func TestConfigurationEdgeCases(t *testing.T) {
	configManager := config.NewConfigManager()
	tempDir := t.TempDir()
	
	edgeCases := []struct {
		name        string
		config      *config.Config
		expectError bool
		description string
	}{
		{
			name: "minimum valid config",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 1124},
				Routes: []config.RouteConfig{{Path: "/", Directory: tempDir}},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: false,
			description: "Minimal valid configuration",
		},
		{
			name: "maximum port number",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 65535},
				Routes: []config.RouteConfig{{Path: "/", Directory: tempDir}},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: false,
			description: "Maximum valid port number",
		},
		{
			name: "port too high",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 65536},
				Routes: []config.RouteConfig{{Path: "/", Directory: tempDir}},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: true,
			description: "Port number too high",
		},
		{
			name: "auth enabled with credentials",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 1124},
				Auth:   config.AuthConfig{Enabled: true, Username: "admin", Password: "secret"},
				Routes: []config.RouteConfig{{Path: "/", Directory: tempDir}},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: false,
			description: "Authentication enabled with valid credentials",
		},
		{
			name: "multiple routes same path",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 1124},
				Routes: []config.RouteConfig{
					{Path: "/static", Directory: tempDir},
					{Path: "/static", Directory: tempDir}, // Duplicate path
				},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: false, // This is allowed, last one wins
			description: "Multiple routes with same path",
		},
		{
			name: "very long path",
			config: &config.Config{
				Server: config.ServerConfig{Host: "localhost", Port: 1124},
				Routes: []config.RouteConfig{{Path: "/" + strings.Repeat("a", 1000), Directory: tempDir}},
				Logging: config.LoggingConfig{Level: "info"},
			},
			expectError: false,
			description: "Very long route path",
		},
	}
	
	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			err := configManager.Validate(tc.config)
			
			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tc.description)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tc.description, err)
			}
		})
	}
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	// Create test environment
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	os.MkdirAll(staticDir, 0755)
	
	// Create test files
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("File %d content", i)
		os.WriteFile(filepath.Join(staticDir, fmt.Sprintf("file%d.txt", i)), []byte(content), 0644)
	}
	
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 0},
		Auth:   config.AuthConfig{Enabled: false},
		Routes: []config.RouteConfig{{Path: "/static", Directory: staticDir}},
		Logging: config.LoggingConfig{Level: "info"},
	}
	
	// Start server
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewBasicAuthenticator(cfg.Auth.Enabled, cfg.Auth.Username, cfg.Auth.Password)
	fileServer := fileserver.NewFileServer()
	httpServer := server.NewHTTPServer(cfg, log, authenticator, fileServer)
	lifecycleManager := server.NewLifecycleManager(httpServer, log)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- lifecycleManager.Run(ctx)
	}()
	
	time.Sleep(200 * time.Millisecond)
	
	baseURL := fmt.Sprintf("http://%s", httpServer.GetAddr())
	
	// Make concurrent requests
	const numRequests = 50
	results := make(chan error, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(fileNum int) {
			url := fmt.Sprintf("%s/static/file%d.txt", baseURL, fileNum%10)
			resp, err := http.Get(url)
			if err != nil {
				results <- fmt.Errorf("request failed: %w", err)
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != 200 {
				results <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- fmt.Errorf("failed to read body: %w", err)
				return
			}
			
			expected := fmt.Sprintf("File %d content", fileNum%10)
			if string(body) != expected {
				results <- fmt.Errorf("unexpected content: got %s, want %s", string(body), expected)
				return
			}
			
			results <- nil
		}(i)
	}
	
	// Collect results
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
	
	cancel()
	
	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("Server returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}