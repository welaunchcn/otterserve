package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Routes  []RouteConfig `yaml:"routes"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// RouteConfig defines a route mapping
type RouteConfig struct {
	Path      string `yaml:"path"`
	Directory string `yaml:"directory"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// ConfigManager interface defines configuration management operations
type ConfigManager interface {
	Load(filename string) (*Config, error)
	Save(config *Config, filename string) error
	Validate(config *Config) error
	LoadOrCreateDefault(filename string) (*Config, error)
}

// DefaultConfigManager implements ConfigManager
type DefaultConfigManager struct{}

// NewConfigManager creates a new configuration manager
func NewConfigManager() ConfigManager {
	return &DefaultConfigManager{}
}

// Load reads and parses a YAML configuration file
func (cm *DefaultConfigManager) Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

// Save writes configuration to a YAML file
func (cm *DefaultConfigManager) Save(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfig returns a configuration with default values
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 1123,
		},
		Auth: AuthConfig{
			Enabled:  false,
			Username: "",
			Password: "",
		},
		Routes: []RouteConfig{
			{Path: "/", Directory: "./"},
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "", // Empty means stdout/stderr
		},
	}
}

// LoadOrCreateDefault loads configuration from file, or creates default if file doesn't exist
func (cm *DefaultConfigManager) LoadOrCreateDefault(filename string) (*Config, error) {
	// Try to load existing config
	config, err := cm.Load(filename)
	if err != nil {
		// If file doesn't exist, create default config
		if errors.Is(err, os.ErrNotExist) {
			config = GetDefaultConfig()

			// Apply defaults to ensure all fields are set
			cm.applyDefaults(config)

			// Create default directories if they don't exist
			if err := cm.createDefaultDirectories(config); err != nil {
				return nil, fmt.Errorf("failed to create default directories: %w", err)
			}

			// Save default config to file
			if err := cm.Save(config, filename); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}

			return config, nil
		}
		return nil, err
	}

	// Apply defaults to loaded config for any missing fields
	cm.applyDefaults(config)

	return config, nil
}

// applyDefaults fills in any missing configuration values with defaults
func (cm *DefaultConfigManager) applyDefaults(config *Config) {
	defaults := GetDefaultConfig()

	// Apply server defaults
	if config.Server.Host == "" {
		config.Server.Host = defaults.Server.Host
	}
	// Note: Port 0 is valid (means "use any available port"), so we don't override it
	// Only apply default port if port is negative (which would be invalid)
	if config.Server.Port < 0 {
		config.Server.Port = defaults.Server.Port
	}

	// Apply logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = defaults.Logging.Level
	}

	// Ensure at least one route exists
	if len(config.Routes) == 0 {
		config.Routes = defaults.Routes
	}
}

// createDefaultDirectories creates the directories referenced in default routes
func (cm *DefaultConfigManager) createDefaultDirectories(config *Config) error {
	for _, route := range config.Routes {
		if err := os.MkdirAll(route.Directory, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", route.Directory, err)
		}
	}
	return nil
}

// Validate checks if the configuration is valid
func (cm *DefaultConfigManager) Validate(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate server configuration
	if config.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}
	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 0 and 65535, got %d", config.Server.Port)
	}

	// Validate authentication configuration
	if config.Auth.Enabled {
		if config.Auth.Username == "" {
			return fmt.Errorf("auth username cannot be empty when auth is enabled")
		}
		if config.Auth.Password == "" {
			return fmt.Errorf("auth password cannot be empty when auth is enabled")
		}
	}

	// Validate routes
	if len(config.Routes) == 0 {
		return fmt.Errorf("at least one route must be configured")
	}

	for i, route := range config.Routes {
		if route.Path == "" {
			return fmt.Errorf("route %d: path cannot be empty", i)
		}
		if route.Directory == "" {
			return fmt.Errorf("route %d: directory cannot be empty", i)
		}

		// Check if directory exists
		if _, err := os.Stat(route.Directory); os.IsNotExist(err) {
			return fmt.Errorf("route %d: directory %s does not exist", i, route.Directory)
		}
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level %s, must be one of: debug, info, warn, error", config.Logging.Level)
	}

	return nil
}
