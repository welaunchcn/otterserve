package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

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
	name    string
}

// ServiceProgram implements the service.Interface for kardianos/service
type ServiceProgram struct {
	configPath string
	logger     logger.Logger
	server     *server.LifecycleManager
	ctx        context.Context
	cancel     context.CancelFunc
	startupLogFile *os.File
	serviceLogFile *os.File
}

// NewServiceManager creates a new service manager
func NewServiceManager(serviceName, displayName, description, configPath string, log logger.Logger) (ServiceManager, error) {
	program := &ServiceProgram{
		configPath: configPath,
		logger:     log,
	}

	// Get the working directory for the service
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Configure service with Windows-specific options
	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      displayName,
		Description:      description,
		WorkingDirectory: exeDir, // Set working directory to the executable's directory
		Arguments:        []string{"-config", configPath}, // Pass config path as argument
		
		// Windows-specific service configuration
		Option: map[string]interface{}{
			// Service start type (auto, manual, disabled)
			"StartType":              "automatic",
			
			// Recovery actions on service failure
			"OnFailure":              "restart",
			"OnFailureDelayDuration": 5 * time.Second,
			"OnFailureResetPeriod":   60,
			
			// Windows service account (LocalSystem has full system access)
			"UserName":              "LocalSystem",
			
			// Ensure the service has access to the desktop (for debugging)
			"Interactive":           false,
			
			// Set the service to auto-restart on crash
			"DelayedAutoStart":      true,
			
			// Set the service to restart after a crash
			"RecoveryAction":        1, // 1 = Restart the service
			"RebootMessage":        fmt.Sprintf("%s service crashed and will restart", serviceName),
		},
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
		name:    serviceName,
	}, nil
}

// Install installs the service
func (sm *SystemServiceManager) Install() error {
	sm.logger.Info("Starting service installation")

	// Get the full path to the executable
	exePath, err := os.Executable()
	if err != nil {
		sm.logger.Error("Failed to get executable path", logger.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Verify the executable exists and is accessible
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("Executable not found at %s", exePath)
		sm.logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	sm.logger.Info("Installing service with executable", logger.Fields{
		"path": exePath,
	})

	// Best-effort: ensure previous instance is fully deleted to avoid MARKED_FOR_DELETE
	_ = waitForServiceDeletion(sm.logger, sm.name, 5*time.Second)

	// Install the service
	err = sm.service.Install()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to install service: %v", err)
		sm.logger.Error(errMsg)
		
		// Try to get more detailed error information
		if exitErr, ok := err.(*exec.ExitError); ok {
			sm.logger.Error("Service installation failed with exit code", logger.Fields{
				"exit_code": exitErr.ExitCode(),
				"stderr":    string(exitErr.Stderr),
			})
		}
		// If possibly marked for deletion, wait and retry once
		if strings.Contains(strings.ToLower(err.Error()), "marked for deletion") {
			sm.logger.Warn("Service marked for deletion, waiting before retry")
			_ = waitForServiceDeletion(sm.logger, sm.name, 15*time.Second)
			if retryErr := sm.service.Install(); retryErr == nil {
				sm.logger.Info("Service installed successfully after retry")
				return nil
			} else {
				sm.logger.Error("Retry install failed", logger.Fields{"error": retryErr.Error()})
			}
		}
		return fmt.Errorf("%s", errMsg)
	}

	sm.logger.Info("Service installed successfully")
	
	// On Windows, we need to ensure the service has the right permissions
	runtime.GC() // Force garbage collection to free resources
	
	// Log success with path information
	exePath, exeErr := os.Executable()
	if exeErr != nil {
		sm.logger.Warn("Failed to get executable path for logging", logger.Fields{
			"error": exeErr.Error(),
		})
	} else {
		sm.logger.Info("Service installation completed successfully", logger.Fields{
			"executable_path": exePath,
			"working_dir":     filepath.Dir(exePath),
		})
	}
	
	return nil
}

// Uninstall uninstalls the service
func (sm *SystemServiceManager) Uninstall() error {
  sm.logger.Info("Uninstalling service")
  
  // Try to stop first (ignore errors)
  _ = sm.service.Stop()

  if err := sm.service.Uninstall(); err != nil {
    sm.logger.Error("Failed to uninstall service", logger.Fields{
      "error": err.Error(),
    })
    return fmt.Errorf("failed to uninstall service: %w", err)
  }

  sm.logger.Info("Service uninstalled successfully")
  // Wait until SCM fully removes the service to avoid MARKED_FOR_DELETE state
  if err := waitForServiceDeletion(sm.logger, sm.name, 30*time.Second); err != nil {
    sm.logger.Warn("Service deletion not yet finalized", logger.Fields{"service": sm.name, "error": err.Error()})
  }
  return nil
}

// waitForServiceDeletion polls SCM via sc.exe until the service no longer exists or timeout
func waitForServiceDeletion(log logger.Logger, name string, timeout time.Duration) error {
  deadline := time.Now().Add(timeout)
  for {
    // sc.exe query <name> returns exit code 1060 if service doesn't exist
    cmd := exec.Command("sc.exe", "query", name)
    if err := cmd.Run(); err != nil {
      // Check exit code
      if exitErr, ok := err.(*exec.ExitError); ok {
        if exitErr.ExitCode() == 1060 { // ERROR_SERVICE_DOES_NOT_EXIST
          return nil
        }
      }
      // Other error; log and continue a bit
      log.Warn("sc query failed", logger.Fields{"error": err.Error()})
    } else {
      // Service still exists (could be marked for delete). Keep waiting.
    }

    if time.Now().After(deadline) {
      return fmt.Errorf("timeout waiting for service '%s' to be deleted", name)
    }
    time.Sleep(500 * time.Millisecond)
  }
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
  // Return quickly to SCM; do heavy init in background
  go sp.startAsync()
  return nil
}

// startAsync performs initialization and starts the server in background
func (sp *ServiceProgram) startAsync() {
  // Create a log file in the system temp directory first
  logPath := filepath.Join(os.TempDir(), "otterserve_service_startup.log")
  lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
  if err == nil {
    sp.startupLogFile = lf
    // In service context, avoid Stdout/Stderr to prevent redirection to system directories
    sp.logger = logger.NewLogger(logger.DebugLevel, lf)
    sp.logger.Info("=== Service Starting - Initial Log ===")
  } else {
    // Last resort: still avoid Stdout/Stderr; use an in-memory pipe fallback to drop messages
    // but inform via event log equivalent message
    sp.logger = logger.NewLogger(logger.DebugLevel, io.Discard)
    // We cannot safely write to stderr in services; just note in memory
  }

  // Log basic information about the service start
  sp.logger.Info("Service program starting")

  // Get and log working directory
  wd, wderr := os.Getwd()
  if wderr != nil {
    sp.logger.Warn("Failed to get working directory", logger.Fields{"error": wderr})
    wd = fmt.Sprintf("[error: %v]", wderr)
  }

  // Log process information
  sp.logger.Info("Process information", logger.Fields{
    "pid":        os.Getpid(),
    "ppid":       os.Getppid(),
    "executable": os.Args[0],
    "args":       os.Args[1:],
    "cwd":        wd,
  })

  // Get the executable's directory to ensure we can find config files and logs
  exePath, err := os.Executable()
  if err != nil {
    sp.logger.Error("Failed to get executable path", logger.Fields{"error": err})
    return
  }
  exeDir := filepath.Dir(exePath)

  // Change to the executable's directory to ensure relative paths work
  if chdirErr := os.Chdir(exeDir); chdirErr != nil {
    sp.logger.Error("Failed to change working directory", logger.Fields{"error": chdirErr, "dir": exeDir})
    return
  }

  // Set up the main service log file beside the executable (not system32)
  serviceLogPath := filepath.Join(exeDir, "otterserve_service.log")
  slf, slfErr := os.OpenFile(serviceLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
  if slfErr != nil {
    sp.logger.Warn("Could not create main log file, using startup logger only",
      logger.Fields{"path": serviceLogPath, "error": slfErr})
  } else {
    sp.serviceLogFile = slf
    // If we can create the log file, update the logger to use file only (no Stdout)
    sp.logger = logger.NewLogger(logger.DebugLevel, slf)
    sp.logger.Info("Created main service log file", logger.Fields{"path": serviceLogPath})
  }

  // Log environment variables (for debugging)
  envVars := []string{"PATH", "SYSTEMROOT", "TEMP", "TMP", "USERNAME", "USERPROFILE", "SYSTEMDRIVE", "WINDIR"}
  envFields := make(logger.Fields)
  for _, v := range envVars {
    envFields[v] = os.Getenv(v)
  }
  sp.logger.Info("Environment variables", envFields)

  // If config path is relative, make it absolute relative to the executable
  if !filepath.IsAbs(sp.configPath) {
    sp.configPath = filepath.Join(exeDir, sp.configPath)
  }

  // Create context for graceful shutdown
  sp.ctx, sp.cancel = context.WithCancel(context.Background())

  // Load configuration
  configManager := config.NewConfigManager()
  cfg, cfgErr := configManager.LoadOrCreateDefault(sp.configPath)
  if cfgErr != nil {
    errMsg := fmt.Sprintf("Failed to load configuration from %s: %v", sp.configPath, cfgErr)
    sp.logger.Error(errMsg)
    return
  }

  // Validate configuration
  if valErr := configManager.Validate(cfg); valErr != nil {
    sp.logger.Error("Configuration validation failed", logger.Fields{
      "error": valErr.Error(),
    })
    return
  }

  // Normalize configured log file: ensure absolute under exeDir when empty or relative
  logFile := cfg.Logging.File
  if logFile == "" {
    logFile = filepath.Join(exeDir, "otterserve.log")
  } else if !filepath.IsAbs(logFile) {
    logFile = filepath.Join(exeDir, logFile)
  }
  // Create logger from configuration (file-backed only in service context)
  appLogger, logErr := logger.NewLoggerFromConfig(cfg.Logging.Level, logFile)
  if logErr != nil {
    sp.logger.Error("Failed to create application logger", logger.Fields{
      "error": logErr.Error(),
    })
    return
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
}

// Stop implements service.Interface - called when service stops
func (sp *ServiceProgram) Stop(s service.Service) error {
  sp.logger.Info("Service program stopping")
  
  if sp.cancel != nil {
    sp.cancel()
  }

  sp.logger.Info("Service program stopped")
  // Close log files after shutdown
  if sp.serviceLogFile != nil {
    _ = sp.serviceLogFile.Close()
    sp.serviceLogFile = nil
  }
  if sp.startupLogFile != nil {
    _ = sp.startupLogFile.Close()
    sp.startupLogFile = nil
  }
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