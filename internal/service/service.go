package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"otterserve/internal/auth"
	"otterserve/internal/config"
	"otterserve/internal/fileserver"
	"otterserve/internal/logger"
	"otterserve/internal/server"
)

// ServiceManager interface defines service management operations
type ServiceManager interface {
	Install() error
	Uninstall() error
	Start() error
	Stop() error
	Run() error
}

// SystemServiceManager implements ServiceManager using kardianos/service
type SystemServiceManager struct {
	service service.Service
	program *ServiceProgram
	logger  logger.Logger
}

// ServiceProgram implements the service.Interface for kardianos/service
type ServiceProgram struct {
	configPath string
	logger     logger.Logger
	server     *server.LifecycleManager
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServiceManager creates a new service manager
func NewServiceManager(serviceName, displayName, description, configPath string, log logger.Logger) (ServiceManager, error) {
	program := &ServiceProgram{
		configPath: configPath,
		logger:     log,
	}

	// Configure service
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: displayName,
		Description: description,
		Arguments:   []string{}, // No additional arguments needed
	}

	// Create service
	svc, err := service.New(program, svcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &SystemServiceManager{
		service: svc,
		program: program,
		logger:  log,
	}, nil
}

// Install installs the service
func (sm *SystemServiceManager) Install() error {
	sm.logger.Info("Installing service")
	
	if err := sm.service.Install(); err != nil {
		sm.logger.Error("Failed to install service", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to install service: %w", err)
	}

	sm.logger.Info("Service installed successfully")
	return nil
}

// Uninstall uninstalls the service
func (sm *SystemServiceManager) Uninstall() error {
	sm.logger.Info("Uninstalling service")
	
	if err := sm.service.Uninstall(); err != nil {
		sm.logger.Error("Failed to uninstall service", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	sm.logger.Info("Service uninstalled successfully")
	return nil
}

// Start starts the service
func (sm *SystemServiceManager) Start() error {
	sm.logger.Info("Starting service")
	
	if err := sm.service.Start(); err != nil {
		sm.logger.Error("Failed to start service", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to start service: %w", err)
	}

	sm.logger.Info("Service started successfully")
	return nil
}

// Stop stops the service
func (sm *SystemServiceManager) Stop() error {
	sm.logger.Info("Stopping service")
	
	if err := sm.service.Stop(); err != nil {
		sm.logger.Error("Failed to stop service", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to stop service: %w", err)
	}

	sm.logger.Info("Service stopped successfully")
	return nil
}

// Run runs the service (blocking call)
func (sm *SystemServiceManager) Run() error {
	sm.logger.Info("Running service")
	
	if err := sm.service.Run(); err != nil {
		sm.logger.Error("Service run failed", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("service run failed: %w", err)
	}

	return nil
}

// Start implements service.Interface - called when service starts
func (sp *ServiceProgram) Start(s service.Service) error {
	sp.logger.Info("Service program starting")
	
	// Create context for graceful shutdown
	sp.ctx, sp.cancel = context.WithCancel(context.Background())
	
	// Load configuration
	configManager := config.NewConfigManager()
	cfg, err := configManager.LoadOrCreateDefault(sp.configPath)
	if err != nil {
		sp.logger.Error("Failed to load configuration", logger.Fields{
			"error": err.Error(),
			"config_path": sp.configPath,
		})
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := configManager.Validate(cfg); err != nil {
		sp.logger.Error("Configuration validation failed", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create logger from configuration
	appLogger, err := logger.NewLoggerFromConfig(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		sp.logger.Error("Failed to create application logger", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to create application logger: %w", err)
	}

	// Create authenticator
	authenticator := auth.NewBasicAuthenticator(
		cfg.Auth.Enabled,
		cfg.Auth.Username,
		cfg.Auth.Password,
	)

	// Create file server
	fileServer := fileserver.NewFileServer()

	// Create HTTP server
	httpServer := server.NewHTTPServer(cfg, appLogger, authenticator, fileServer)

	// Create lifecycle manager
	sp.server = server.NewLifecycleManager(httpServer, appLogger)

	// Start server in goroutine
	go func() {
		if err := sp.server.Run(sp.ctx); err != nil {
			sp.logger.Error("Server lifecycle manager failed", logger.Fields{
				"error": err.Error(),
			})
		}
	}()

	sp.logger.Info("Service program started successfully")
	return nil
}

// Stop implements service.Interface - called when service stops
func (sp *ServiceProgram) Stop(s service.Service) error {
	sp.logger.Info("Service program stopping")
	
	if sp.cancel != nil {
		sp.cancel()
	}

	sp.logger.Info("Service program stopped")
	return nil
}

// ConsoleRunner runs the service in console mode (not as a system service)
type ConsoleRunner struct {
	configPath string
	logger     logger.Logger
}

// NewConsoleRunner creates a new console runner
func NewConsoleRunner(configPath string, log logger.Logger) *ConsoleRunner {
	return &ConsoleRunner{
		configPath: configPath,
		logger:     log,
	}
}

// Run runs the application in console mode
func (cr *ConsoleRunner) Run() error {
	cr.logger.Info("Starting application in console mode")

	// Load configuration
	configManager := config.NewConfigManager()
	cfg, err := configManager.LoadOrCreateDefault(cr.configPath)
	if err != nil {
		cr.logger.Error("Failed to load configuration", logger.Fields{
			"error": err.Error(),
			"config_path": cr.configPath,
		})
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := configManager.Validate(cfg); err != nil {
		cr.logger.Error("Configuration validation failed", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create logger from configuration
	appLogger, err := logger.NewLoggerFromConfig(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		cr.logger.Error("Failed to create application logger", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to create application logger: %w", err)
	}

	// Create authenticator
	authenticator := auth.NewBasicAuthenticator(
		cfg.Auth.Enabled,
		cfg.Auth.Username,
		cfg.Auth.Password,
	)

	// Create file server
	fileServer := fileserver.NewFileServer()

	// Create HTTP server
	httpServer := server.NewHTTPServer(cfg, appLogger, authenticator, fileServer)

	// Create lifecycle manager
	lifecycleManager := server.NewLifecycleManager(httpServer, appLogger)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		cr.logger.Info("Received shutdown signal", logger.Fields{
			"signal": sig.String(),
		})
		cancel()
	}()

	// Run the server
	if err := lifecycleManager.Run(ctx); err != nil {
		cr.logger.Error("Application failed", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	cr.logger.Info("Application shutdown completed")
	return nil
}