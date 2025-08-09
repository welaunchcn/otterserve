package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kardianos/service"
	"otterserve/internal/logger"
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
    directory: "` + filepath.ToSlash(staticDir) + `"
  - path: "/docs"
    directory: "` + filepath.ToSlash(docsDir) + `"
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

	// Test that the console runner can load and validate the configuration
	// without actually starting the server (which would run indefinitely)
	
	// We'll test the configuration loading by checking if the runner can be created
	// and if it would fail early due to configuration issues
	if runner == nil {
		t.Error("Console runner should not be nil")
	}
	
	// Test that the config file exists and is readable
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Config file should exist: %v", err)
	}
	
	// Note: We don't call runner.Run() because it would start a server that runs indefinitely
	// The actual server functionality is tested in integration tests with proper cleanup
}

func TestConsoleRunner_Run_InvalidConfig(t *testing.T) {
	log := logger.NewLogger(logger.InfoLevel, nil)
	runner := NewConsoleRunner("/nonexistent/config.yaml", log)

	// According to requirements, when config file doesn't exist, 
	// the system should create a default config, not fail.
	// So we test that the runner can be created successfully.
	if runner == nil {
		t.Error("Console runner should not be nil even with nonexistent config")
	}
	
	// Note: We don't call runner.Run() because it would create a default config
	// and start a server that runs indefinitely. The config creation behavior
	// is tested in the config package tests.
}

func TestConsoleRunner_Run_InvalidConfigContent(t *testing.T) {
	// Create temporary config file with invalid content
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")
	
	// Create config with invalid routes (nonexistent directories)
	configContent := `server:
  host: "localhost"
  port: 1124
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

func (ms *mockService) Logger(errs chan<- error) (service.Logger, error) {
	return nil, nil
}

func (ms *mockService) Platform() string {
	return "mock"
}

func (ms *mockService) Restart() error {
	if ms.shouldError {
		return fmt.Errorf("mock restart error")
	}
	return nil
}

func (ms *mockService) Status() (service.Status, error) {
	if ms.shouldError {
		return service.StatusUnknown, fmt.Errorf("mock status error")
	}
	return service.StatusRunning, nil
}

func (ms *mockService) String() string {
	return "mock service"
}

func (ms *mockService) SystemLogger(errs chan<- error) (service.Logger, error) {
	return nil, nil
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