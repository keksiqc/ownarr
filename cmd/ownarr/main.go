package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/keksiqc/ownarr/internal/config"
	"github.com/keksiqc/ownarr/internal/processor"
	"github.com/keksiqc/ownarr/internal/watcher"
)

const (
	appName    = "ownarr"
	appVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "config.yaml", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	if *showHelp {
		fmt.Printf("%s - A lightweight file watcher and permission manager\n\n", appName)
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Initialize logger with default settings
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Prefix:          appName,
	})

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}

	// Set log level from configuration
	if err := setLogLevel(logger, cfg.LogLevel); err != nil {
		logger.Fatal("Invalid log level", "level", cfg.LogLevel, "error", err)
	}

	logger.Info("Starting application",
		"version", appVersion,
		"config", *configPath,
		"log_level", cfg.LogLevel,
		"poll_interval", cfg.PollInterval,
		"watch_dirs", len(cfg.WatchDirs),
	)

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize watcher
	w, err := watcher.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create watcher", "error", err)
	}
	// Watcher will be closed explicitly in shutdown sequence

	// Initialize processor
	proc := processor.New(logger)

	// Start watching
	if err := w.Start(ctx); err != nil {
		logger.Fatal("Failed to start watcher", "error", err)
	}

	// Start processing events
	go proc.Process(ctx, w.Events(), w.Errors())

	logger.Info("Application started successfully")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Received shutdown signal, stopping...")

	// Cancel context to signal all goroutines to stop
	cancel()

	// Close watcher properly
	if err := w.Close(); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	// Give a moment for cleanup
	time.Sleep(500 * time.Millisecond)

	logger.Info("Application stopped")
}

// setLogLevel sets the logger level based on the configuration
func setLogLevel(logger *log.Logger, level string) error {
	switch level {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "info":
		logger.SetLevel(log.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	case "fatal", "critical":
		logger.SetLevel(log.FatalLevel)
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}
	return nil
}
