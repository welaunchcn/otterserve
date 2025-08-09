package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager_Load(t *testing.T) {
	cm := NewConfigManager()

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	
	configContent := `server:
  host: "localhost"
  port: 1124
auth:
  enabled: true
  username: "admin"
  password: "secret"
routes:
  - path: "/static"
    directory: "./static"
  - path: "/docs"
    directory: "./docs"
logging:
  level: "info"
  file: ""
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading configuration
	config, err := cm.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 1124 {
		t.Errorf("Expected port 1124, got %d", config.Server.Port)
	}
	if !config.Auth.Enabled {
		t.Error("Expected auth to be enabled")
	}
	if config.Auth.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", config.Auth.Username)
	}
	if config.Auth.Password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", config.Auth.Password)
	}
	if len(config.Routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(config.Routes))
	}
	if config.Routes[0].Path != "/static" {
		t.Errorf("Expected first route path '/static', got '%s'", config.Routes[0].Path)
	}
	if config.Routes[0].Directory != "./static" {
		t.Errorf("Expected first route directory './static', got '%s'", config.Routes[0].Directory)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Logging.Level)
	}
}

func TestConfigManager_Load_FileNotFound(t *testing.T) {
	cm := NewConfigManager()
	
	_, err := cm.Load("nonexistent-file.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestConfigManager_Load_InvalidYAML(t *testing.T) {
	cm := NewConfigManager()

	// Create a temporary file with invalid YAML
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")
	
	invalidYAML := `server:
  host: "localhost"
  port: invalid_port
auth:
  enabled: not_a_boolean
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = cm.Load(configFile)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestConfigManager_Save(t *testing.T) {
	cm := NewConfigManager()

	config := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
		Auth: AuthConfig{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		Routes: []RouteConfig{
			{Path: "/files", Directory: "./files"},
		},
		Logging: LoggingConfig{
			Level: "debug",
			File:  "/var/log/service.log",
		},
	}

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "saved-config.yaml")

	// Test saving configuration
	err := cm.Save(config, configFile)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load the saved config and verify
	loadedConfig, err := cm.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Server.Host != config.Server.Host {
		t.Errorf("Expected host '%s', got '%s'", config.Server.Host, loadedConfig.Server.Host)
	}
	if loadedConfig.Server.Port != config.Server.Port {
		t.Errorf("Expected port %d, got %d", config.Server.Port, loadedConfig.Server.Port)
	}
}

func TestConfigManager_Validate(t *testing.T) {
	cm := NewConfigManager()

	// Create test directories
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	docsDir := filepath.Join(tempDir, "docs")
	os.MkdirAll(staticDir, 0755)
	os.MkdirAll(docsDir, 0755)

	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty host",
			config: &Config{
				Server: ServerConfig{Host: "", Port: 1124},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: -1},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "auth enabled without username",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: true, Username: "", Password: "secret"},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "auth enabled without password",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: true, Username: "admin", Password: ""},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "no routes",
			config: &Config{
				Server:  ServerConfig{Host: "localhost", Port: 1124},
				Auth:    AuthConfig{Enabled: false},
				Routes:  []RouteConfig{},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "route with empty path",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "route with nonexistent directory",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "/static", Directory: "/nonexistent/directory"},
				},
				Logging: LoggingConfig{Level: "info"},
			},
			expectError: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Server: ServerConfig{Host: "localhost", Port: 1124},
				Auth:   AuthConfig{Enabled: false},
				Routes: []RouteConfig{
					{Path: "/static", Directory: staticDir},
				},
				Logging: LoggingConfig{Level: "invalid"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Validate(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}
func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// Verify default values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 1123 {
		t.Errorf("Expected default port 1123, got %d", config.Server.Port)
	}
	if config.Auth.Enabled {
		t.Error("Expected auth to be disabled by default")
	}
	if len(config.Routes) != 2 {
		t.Errorf("Expected 2 default routes, got %d", len(config.Routes))
	}
	if config.Routes[0].Path != "/static" || config.Routes[0].Directory != "./static" {
		t.Errorf("Expected first route '/static' -> './static', got '%s' -> '%s'", 
			config.Routes[0].Path, config.Routes[0].Directory)
	}
	if config.Routes[1].Path != "/docs" || config.Routes[1].Directory != "./docs" {
		t.Errorf("Expected second route '/docs' -> './docs', got '%s' -> '%s'", 
			config.Routes[1].Path, config.Routes[1].Directory)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", config.Logging.Level)
	}
	if config.Logging.File != "" {
		t.Errorf("Expected default log file to be empty, got '%s'", config.Logging.File)
	}
}

func TestConfigManager_LoadOrCreateDefault_CreateNew(t *testing.T) {
	cm := NewConfigManager()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "new-config.yaml")

	// Change to temp directory so relative paths work
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// File doesn't exist, should create default
	config, err := cm.LoadOrCreateDefault(configFile)
	if err != nil {
		t.Fatalf("Failed to load or create default config: %v", err)
	}

	// Verify it's the default config
	defaultConfig := GetDefaultConfig()
	if config.Server.Host != defaultConfig.Server.Host {
		t.Errorf("Expected host '%s', got '%s'", defaultConfig.Server.Host, config.Server.Host)
	}
	if config.Server.Port != defaultConfig.Server.Port {
		t.Errorf("Expected port %d, got %d", defaultConfig.Server.Port, config.Server.Port)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify directories were created
	for _, route := range config.Routes {
		if _, err := os.Stat(route.Directory); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", route.Directory)
		}
	}
}

func TestConfigManager_LoadOrCreateDefault_LoadExisting(t *testing.T) {
	cm := NewConfigManager()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "existing-config.yaml")

	// Create existing config with custom values
	existingConfig := &Config{
		Server: ServerConfig{Host: "0.0.0.0", Port: 9090},
		Auth:   AuthConfig{Enabled: true, Username: "admin", Password: "secret"},
		Routes: []RouteConfig{
			{Path: "/files", Directory: tempDir}, // Use tempDir as it exists
		},
		Logging: LoggingConfig{Level: "debug", File: "/tmp/service.log"},
	}

	err := cm.Save(existingConfig, configFile)
	if err != nil {
		t.Fatalf("Failed to save existing config: %v", err)
	}

	// Load existing config
	config, err := cm.LoadOrCreateDefault(configFile)
	if err != nil {
		t.Fatalf("Failed to load existing config: %v", err)
	}

	// Verify it loaded the existing config, not defaults
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}
	if !config.Auth.Enabled {
		t.Error("Expected auth to be enabled")
	}
}

func TestConfigManager_ApplyDefaults(t *testing.T) {
	cm := &DefaultConfigManager{}

	// Create config with missing values
	config := &Config{
		Server: ServerConfig{Host: "", Port: -1}, // Missing/invalid values
		Auth:   AuthConfig{Enabled: false},
		Routes: []RouteConfig{}, // Empty routes
		Logging: LoggingConfig{Level: "", File: ""}, // Missing level
	}

	cm.applyDefaults(config)

	// Verify defaults were applied
	if config.Server.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 1123 {
		t.Errorf("Expected default port 1123, got %d", config.Server.Port)
	}
	if len(config.Routes) != 2 {
		t.Errorf("Expected 2 default routes, got %d", len(config.Routes))
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", config.Logging.Level)
	}
}

func TestConfigManager_ApplyDefaults_PreserveExisting(t *testing.T) {
	cm := &DefaultConfigManager{}

	// Create config with existing values
	config := &Config{
		Server: ServerConfig{Host: "custom-host", Port: 9000},
		Auth:   AuthConfig{Enabled: true, Username: "user", Password: "pass"},
		Routes: []RouteConfig{
			{Path: "/custom", Directory: "./custom"},
		},
		Logging: LoggingConfig{Level: "debug", File: "/custom/log"},
	}

	cm.applyDefaults(config)

	// Verify existing values were preserved
	if config.Server.Host != "custom-host" {
		t.Errorf("Expected preserved host 'custom-host', got '%s'", config.Server.Host)
	}
	if config.Server.Port != 9000 {
		t.Errorf("Expected preserved port 9000, got %d", config.Server.Port)
	}
	if len(config.Routes) != 1 {
		t.Errorf("Expected 1 preserved route, got %d", len(config.Routes))
	}
	if config.Routes[0].Path != "/custom" {
		t.Errorf("Expected preserved route path '/custom', got '%s'", config.Routes[0].Path)
	}
	if config.Logging.Level != "debug" {
		t.Errorf("Expected preserved log level 'debug', got '%s'", config.Logging.Level)
	}
}

func TestConfigManager_CreateDefaultDirectories(t *testing.T) {
	cm := &DefaultConfigManager{}
	tempDir := t.TempDir()

	config := &Config{
		Routes: []RouteConfig{
			{Path: "/test1", Directory: filepath.Join(tempDir, "test1")},
			{Path: "/test2", Directory: filepath.Join(tempDir, "test2", "subdir")},
		},
	}

	err := cm.createDefaultDirectories(config)
	if err != nil {
		t.Fatalf("Failed to create default directories: %v", err)
	}

	// Verify directories were created
	for _, route := range config.Routes {
		if _, err := os.Stat(route.Directory); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", route.Directory)
		}
	}
}