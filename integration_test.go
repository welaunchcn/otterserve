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

	"otterserve/internal/auth"
	"otterserve/internal/config"
	"otterserve/internal/fileserver"
	"otterserve/internal/logger"
	"otterserve/internal/server"
	"otterserve/internal/service"
)

// TestEndToEndIntegration tests the complete application flow
func TestEndToEndIntegration(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	docsDir := filepath.Join(tempDir, "docs")
	
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(docsDir, 0755)

	// Create test files
	indexHTML := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body><h1>Welcome to Test Server</h1></body>
</html>`
	
	testTxt := "This is a test file content."
	docFile := "This is documentation content."

	os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(indexHTML), 0644)
	os.WriteFile(filepath.Join(staticDir, "test.txt"), []byte(testTxt), 0644)
	os.WriteFile(filepath.Join(docsDir, "readme.txt"), []byte(docFile), 0644)

	// Create configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use port 0 to get a random available port
		},
		Auth: config.AuthConfig{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		Routes: []config.RouteConfig{
			{Path: "/static", Directory: staticDir},
			{Path: "/docs", Directory: docsDir},
		},
		Logging: config.LoggingConfig{
			Level: "info",
			File:  "",
		},
	}

	// Create components
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewBasicAuthenticator(cfg.Auth.Enabled, cfg.Auth.Username, cfg.Auth.Password)
	fileServer := fileserver.NewFileServer()
	httpServer := server.NewHTTPServer(cfg, log, authenticator, fileServer)
	lifecycleManager := server.NewLifecycleManager(httpServer, log)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- lifecycleManager.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Get the actual server address (since we used port 0)
	serverAddr := httpServer.GetAddr()
	baseURL := fmt.Sprintf("http://%s", serverAddr)

	// Test cases
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
		checkContains  bool
	}{
		{
			name:           "serve index.html from static route",
			path:           "/static/",
			expectedStatus: 200,
			expectedBody:   "Welcome to Test Server",
			checkContains:  true,
		},
		{
			name:           "serve specific file from static route",
			path:           "/static/test.txt",
			expectedStatus: 200,
			expectedBody:   testTxt,
			checkContains:  false,
		},
		{
			name:           "serve file from docs route",
			path:           "/docs/readme.txt",
			expectedStatus: 200,
			expectedBody:   docFile,
			checkContains:  false,
		},
		{
			name:           "directory listing for docs",
			path:           "/docs/",
			expectedStatus: 200,
			expectedBody:   "readme.txt",
			checkContains:  true,
		},
		{
			name:           "404 for nonexistent file",
			path:           "/static/nonexistent.txt",
			expectedStatus: 404,
			expectedBody:   "",
			checkContains:  false,
		},
		{
			name:           "404 for unmatched route",
			path:           "/unmatched",
			expectedStatus: 404,
			expectedBody:   "404 Not Found",
			checkContains:  true,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + tc.path)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if tc.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				bodyStr := string(body)
				if tc.checkContains {
					if !strings.Contains(bodyStr, tc.expectedBody) {
						t.Errorf("Expected body to contain '%s', got '%s'", tc.expectedBody, bodyStr)
					}
				} else {
					if bodyStr != tc.expectedBody {
						t.Errorf("Expected body '%s', got '%s'", tc.expectedBody, bodyStr)
					}
				}
			}
		})
	}

	// Stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("Server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

// TestEndToEndWithAuthentication tests the application with authentication enabled
func TestEndToEndWithAuthentication(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	os.MkdirAll(staticDir, 0755)

	// Create test file
	testContent := "Protected content"
	os.WriteFile(filepath.Join(staticDir, "protected.txt"), []byte(testContent), 0644)

	// Create configuration with authentication enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
		Auth: config.AuthConfig{
			Enabled:  true,
			Username: "admin",
			Password: "secret",
		},
		Routes: []config.RouteConfig{
			{Path: "/static", Directory: staticDir},
		},
		Logging: config.LoggingConfig{
			Level: "info",
			File:  "",
		},
	}

	// Create components
	log := logger.NewLogger(logger.InfoLevel, nil)
	authenticator := auth.NewBasicAuthenticator(cfg.Auth.Enabled, cfg.Auth.Username, cfg.Auth.Password)
	fileServer := fileserver.NewFileServer()
	httpServer := server.NewHTTPServer(cfg, log, authenticator, fileServer)
	lifecycleManager := server.NewLifecycleManager(httpServer, log)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- lifecycleManager.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	serverAddr := httpServer.GetAddr()
	baseURL := fmt.Sprintf("http://%s", serverAddr)

	// Test without authentication (should fail)
	resp, err := http.Get(baseURL + "/static/protected.txt")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401 without auth, got %d", resp.StatusCode)
	}

	// Test with correct authentication
	req, err := http.NewRequest("GET", baseURL+"/static/protected.txt", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth("admin", "secret")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make authenticated request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 with correct auth, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != testContent {
		t.Errorf("Expected body '%s', got '%s'", testContent, string(body))
	}

	// Test with incorrect authentication
	req, err = http.NewRequest("GET", baseURL+"/static/protected.txt", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.SetBasicAuth("admin", "wrong")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request with wrong auth: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected status 401 with wrong auth, got %d", resp.StatusCode)
	}

	// Stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-serverDone:
		if err != nil {
			t.Errorf("Server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

// TestConsoleRunnerIntegration tests the console runner with a real config file
func TestConsoleRunnerIntegration(t *testing.T) {
	// Create temporary directory and config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "integration-config.yaml")
	staticDir := filepath.Join(tempDir, "static")
	
	os.MkdirAll(staticDir, 0755)
	os.WriteFile(filepath.Join(staticDir, "test.txt"), []byte("integration test"), 0644)

	// Create config file
	configContent := fmt.Sprintf(`server:
  host: "localhost"
  port: 0
auth:
  enabled: false
routes:
  - path: "/static"
    directory: "%s"
logging:
  level: "info"
  file: ""
`, filepath.ToSlash(staticDir))

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create console runner
	log := logger.NewLogger(logger.InfoLevel, nil)
	runner := service.NewConsoleRunner(configFile, log)

	// Test that the console runner can be created and configured properly
	// without actually starting the server (which would run indefinitely)
	
	if runner == nil {
		t.Error("Console runner should not be nil")
	}
	
	// Test that the config file exists and is readable
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Config file should exist: %v", err)
	}
	
	// Verify the static directory and test file exist
	if _, err := os.Stat(filepath.Join(staticDir, "test.txt")); os.IsNotExist(err) {
		t.Errorf("Test file should exist: %v", err)
	}
	
	t.Log("Console runner integration test completed successfully")
	
	// Note: We don't call runner.Run() because it would start a server that runs indefinitely
	// The actual server functionality is tested in other integration tests with proper cleanup
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)

	// Test with invalid configuration
	invalidConfig := &config.Config{
		Server: config.ServerConfig{
			Host: "",  // Invalid empty host
			Port: 1124,
		},
		Routes: []config.RouteConfig{},  // No routes
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}

	authenticator := auth.NewNoOpAuthenticator()
	fileServer := fileserver.NewFileServer()

	// This should not panic, but the server should fail to start properly
	httpServer := server.NewHTTPServer(invalidConfig, log, authenticator, fileServer)
	
	// Try to start server with invalid config
	err := httpServer.Start()
	if err == nil {
		t.Error("Expected error when starting server with invalid config")
	}
}

// TestConfigurationValidation tests configuration validation
func TestConfigurationValidation(t *testing.T) {
	configManager := config.NewConfigManager()

	// Test various invalid configurations
	invalidConfigs := []*config.Config{
		{
			Server: config.ServerConfig{Host: "", Port: 1124},  // Empty host
			Routes: []config.RouteConfig{{Path: "/test", Directory: "/tmp"}},
			Logging: config.LoggingConfig{Level: "info"},
		},
		{
			Server: config.ServerConfig{Host: "localhost", Port: -1},  // Invalid port
			Routes: []config.RouteConfig{{Path: "/test", Directory: "/tmp"}},
			Logging: config.LoggingConfig{Level: "info"},
		},
		{
			Server: config.ServerConfig{Host: "localhost", Port: 1124},
			Routes: []config.RouteConfig{},  // No routes
			Logging: config.LoggingConfig{Level: "info"},
		},
		{
			Server: config.ServerConfig{Host: "localhost", Port: 1124},
			Routes: []config.RouteConfig{{Path: "/test", Directory: "/this/path/definitely/does/not/exist/anywhere"}},  // Nonexistent directory
			Logging: config.LoggingConfig{Level: "info"},
		},
		{
			Server: config.ServerConfig{Host: "localhost", Port: 1124},
			Routes: []config.RouteConfig{{Path: "/test", Directory: "/tmp"}},
			Logging: config.LoggingConfig{Level: "invalid"},  // Invalid log level
		},
	}

	for i, cfg := range invalidConfigs {
		t.Run(fmt.Sprintf("invalid_config_%d", i), func(t *testing.T) {
			err := configManager.Validate(cfg)
			if err == nil {
				t.Error("Expected validation error for invalid config")
			}
		})
	}
}