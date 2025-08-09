package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mini-http-service/internal/logger"
)

func TestNewServiceManager(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	
	sm, err := NewServiceManager("test-service", "Test Service", "Test Description", "config.yaml", log)
	if err != nil {
		t.Fatalf("Failed to create service manager: %v", err)
	}

	if sm == nil {
		t.Fatal("Expected service manager to be created")
	}

	// Verify it's the correct type
	_, ok := sm.(*SystemServiceManager)
	if !ok {
		t.Error("Expected SystemServiceManager type")
	}
}

func TestNewConsoleRunner(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	
	runner := NewConsoleRunner("config.yaml", log)
	if runner == nil {
		t.Fatal("Expected console runner to be created")
	}

	if runner.configPath != "config.yaml" {
		t.Errorf("Expected config path 'config.yaml', got '%s'", runner.configPath)
	}
}

func TestConsoleRunner_Run(t *testing.T) {
	// Create temporary directory and config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	
	// Create test directories
	staticDir := filepath.Join(tempDir, "static")
	docsDir := filepath.Join(tempDir, "docs")
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(docsDir, 0755)

	// Create a basic config file
	configContent := `server:
  host: "localhost"
  port: 0
auth:
  enabled: false
routes:
  - path: "/static"
    directory: "` + staticDir + `"
  - path: "/docs"
    directory: "` + docsDir + `"
logging:
  level: "info"
  file: ""
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	log := logger.NewLogger(logger.InfoLevel, nil)
	runner := NewConsoleRunner(configFile, log)

	// Create context that will be cancelled to stop the runner
	ctx, cancel := context.WithCancel(context.Background())

	// Run in goroutine
	done := make(chan error, 1)
	go func() {
		// We need to simulate the signal handling since we can't easily test it
		// Instead, we'll cancel the context after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		
		done <- runner.Run()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Console runner returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Console runner did not complete within timeout")
		cancel()
	}
}

func TestConsoleRunner_Run_InvalidConfig(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	runner := NewConsoleRunner("/nonexistent/config.yaml", log)

	err := runner.Run()
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestConsoleRunner_Run_InvalidConfigContent(t *testing.T) {
	// Create temporary config file with invalid content
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")
	
	// Create config with invalid routes (nonexistent directories)
	configContent := `server:
  host: "localhost"
  port: 8080
auth:
  enabled: false
routes:
  - path: "/static"
    directory: "/nonexistent/directory"
logging:
  level: "info"
  file: ""
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	log := logger.NewLogger(logger.InfoLevel, nil)
	runner := NewConsoleRunner(configFile, log)

	err = runner.Run()
	if err == nil {
		t.Error("Expected error for invalid config content")
	}
}

// Mock service for testing service manager operations
type mockService struct {
	installCalled   bool
	uninstallCalled bool
	startCalled     bool
	stopCalled      bool
	runCalled       bool
	shouldError     bool
}

func (ms *mockService) Install() error {
	ms.installCalled = true
	if ms.shouldError {
		return fmt.Errorf("mock install error")
	}
	return nil
}

func (ms *mockService) Uninstall() error {
	ms.uninstallCalled = true
	if ms.shouldError {
		return fmt.Errorf("mock uninstall error")
	}
	return nil
}

func (ms *mockService) Start() error {
	ms.startCalled = true
	if ms.shouldError {
		return fmt.Errorf("mock start error")
	}
	return nil
}

func (ms *mockService) Stop() error {
	ms.stopCalled = true
	if ms.shouldError {
		return fmt.Errorf("mock stop error")
	}
	return nil
}

func (ms *mockService) Run() error {
	ms.runCalled = true
	if ms.shouldError {
		return fmt.Errorf("mock run error")
	}
	return nil
}

func TestSystemServiceManager_Operations(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	mockSvc := &mockService{}
	
	sm := &SystemServiceManager{
		service: mockSvc,
		logger:  log,
	}

	// Test Install
	err := sm.Install()
	if err != nil {
		t.Errorf("Install failed: %v", err)
	}
	if !mockSvc.installCalled {
		t.Error("Expected Install to be called on underlying service")
	}

	// Test Uninstall
	err = sm.Uninstall()
	if err != nil {
		t.Errorf("Uninstall failed: %v", err)
	}
	if !mockSvc.uninstallCalled {
		t.Error("Expected Uninstall to be called on underlying service")
	}

	// Test Start
	err = sm.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
	if !mockSvc.startCalled {
		t.Error("Expected Start to be called on underlying service")
	}

	// Test Stop
	err = sm.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if !mockSvc.stopCalled {
		t.Error("Expected Stop to be called on underlying service")
	}

	// Test Run
	err = sm.Run()
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
	if !mockSvc.runCalled {
		t.Error("Expected Run to be called on underlying service")
	}
}

func TestSystemServiceManager_OperationsWithErrors(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	mockSvc := &mockService{shouldError: true}
	
	sm := &SystemServiceManager{
		service: mockSvc,
		logger:  log,
	}

	// Test Install error
	err := sm.Install()
	if err == nil {
		t.Error("Expected Install to return error")
	}

	// Test Uninstall error
	err = sm.Uninstall()
	if err == nil {
		t.Error("Expected Uninstall to return error")
	}

	// Test Start error
	err = sm.Start()
	if err == nil {
		t.Error("Expected Start to return error")
	}

	// Test Stop error
	err = sm.Stop()
	if err == nil {
		t.Error("Expected Stop to return error")
	}

	// Test Run error
	err = sm.Run()
	if err == nil {
		t.Error("Expected Run to return error")
	}
}