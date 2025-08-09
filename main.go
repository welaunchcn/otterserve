package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"mini-http-service/internal/config"
	"mini-http-service/internal/logger"
	"mini-http-service/internal/service"
)

const (
	serviceName    = "mini-http-service"
	serviceDisplay = "Mini HTTP Service"
	serviceDesc    = "Lightweight HTTP file server with configurable routing"
	version        = "1.0.0"
	defaultConfig  = "config.yaml"
)

func main() {
	var (
		install    = flag.Bool("install", false, "Install the service")
		uninstall  = flag.Bool("uninstall", false, "Uninstall the service")
		help       = flag.Bool("help", false, "Show help information")
		showVer    = flag.Bool("version", false, "Show version information")
		configPath = flag.String("config", defaultConfig, "Path to configuration file")
	)
	
	flag.Parse()

	// Create logger for CLI operations
	log := logger.NewLogger(logger.InfoLevel, os.Stdout)

	// Show version information
	if *showVer {
		fmt.Printf("%s version %s\n", serviceName, version)
		return
	}

	// Show help information
	if *help {
		showHelp()
		return
	}

	// Get absolute path for config file
	absConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		log.Error("Failed to get absolute config path", logger.Fields{
			"error": err.Error(),
			"path":  *configPath,
		})
		os.Exit(1)
	}

	// Handle service installation
	if *install {
		if err := installService(absConfigPath, log); err != nil {
			log.Error("Service installation failed", logger.Fields{
				"error": err.Error(),
			})
			os.Exit(1)
		}
		return
	}

	// Handle service uninstallation
	if *uninstall {
		if err := uninstallService(absConfigPath, log); err != nil {
			log.Error("Service uninstallation failed", logger.Fields{
				"error": err.Error(),
			})
			os.Exit(1)
		}
		return
	}

	// Default: run the service in console mode
	if err := runConsole(absConfigPath, log); err != nil {
		log.Error("Application failed", logger.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

// installService installs the service
func installService(configPath string, log logger.Logger) error {
	log.Info("Installing service", logger.Fields{
		"service": serviceDisplay,
		"config":  configPath,
	})

	// Validate that config file exists and is readable
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Warn("Configuration file does not exist, will be created with defaults", logger.Fields{
			"config": configPath,
		})
	} else if err != nil {
		return fmt.Errorf("cannot access configuration file %s: %w", configPath, err)
	}

	// Test configuration loading to catch issues early
	configManager := config.NewConfigManager()
	cfg, err := configManager.LoadOrCreateDefault(configPath)
	if err != nil {
		return fmt.Errorf("failed to load or create configuration: %w", err)
	}

	if err := configManager.Validate(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Info("Configuration validated successfully")

	serviceManager, err := service.NewServiceManager(
		serviceName,
		serviceDisplay,
		serviceDesc,
		configPath,
		log,
	)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	if err := serviceManager.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	log.Info("Service installed successfully", logger.Fields{
		"service": serviceDisplay,
	})
	
	fmt.Printf("%s has been installed successfully.\n", serviceDisplay)
	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Println("You can now start it using your system's service manager.")
	
	return nil
}

// uninstallService uninstalls the service
func uninstallService(configPath string, log logger.Logger) error {
	log.Info("Uninstalling service", logger.Fields{
		"service": serviceDisplay,
	})

	serviceManager, err := service.NewServiceManager(
		serviceName,
		serviceDisplay,
		serviceDesc,
		configPath,
		log,
	)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	if err := serviceManager.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	log.Info("Service uninstalled successfully", logger.Fields{
		"service": serviceDisplay,
	})
	
	fmt.Printf("%s has been uninstalled successfully.\n", serviceDisplay)
	
	return nil
}

// runConsole runs the application in console mode
func runConsole(configPath string, log logger.Logger) error {
	log.Info("Starting application in console mode", logger.Fields{
		"config": configPath,
	})

	// Pre-validate configuration before starting
	configManager := config.NewConfigManager()
	cfg, err := configManager.LoadOrCreateDefault(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := configManager.Validate(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Info("Configuration loaded and validated successfully", logger.Fields{
		"host":   cfg.Server.Host,
		"port":   cfg.Server.Port,
		"routes": len(cfg.Routes),
		"auth":   cfg.Auth.Enabled,
	})

	runner := service.NewConsoleRunner(configPath, log)
	
	fmt.Printf("Starting %s in console mode...\n", serviceDisplay)
	fmt.Printf("Configuration: %s\n", configPath)
	fmt.Printf("Server will listen on: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	if cfg.Auth.Enabled {
		fmt.Println("Authentication: Enabled")
	} else {
		fmt.Println("Authentication: Disabled")
	}
	fmt.Printf("Routes configured: %d\n", len(cfg.Routes))
	for _, route := range cfg.Routes {
		fmt.Printf("  %s -> %s\n", route.Path, route.Directory)
	}
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
	
	return runner.Run()
}

// showHelp displays help information
func showHelp() {
	fmt.Printf("%s - %s\n\n", serviceDisplay, serviceDesc)
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	fmt.Println("  -install           Install the service")
	fmt.Println("  -uninstall         Uninstall the service")
	fmt.Println("  -config <path>     Path to configuration file (default: config.yaml)")
	fmt.Println("  -version           Show version information")
	fmt.Println("  -help              Show this help message")
	fmt.Println()
	fmt.Println("When run without options, the service will start in console mode.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                           # Run in console mode with default config\n", os.Args[0])
	fmt.Printf("  %s -config /path/to/config   # Run with custom config file\n", os.Args[0])
	fmt.Printf("  %s -install                  # Install as system service\n", os.Args[0])
	fmt.Printf("  %s -uninstall                # Uninstall system service\n", os.Args[0])
}