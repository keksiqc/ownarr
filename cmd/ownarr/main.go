package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keksiqc/ownarr/internal/config"
	"github.com/keksiqc/ownarr/internal/enforcer"
	"github.com/keksiqc/ownarr/internal/logger"
	"github.com/keksiqc/ownarr/internal/server"
)

func main() {
	log := logger.New("info")

	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Error("Configuration error")
		os.Exit(1)
	}

	// Update log level based on config
	log = logger.New(cfg.LogLevel)
	log.Info("Starting ownarr",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
		"poll_interval", cfg.PollInterval,
		"timezone", cfg.Timezone.String(),
		"folders", len(cfg.Folders),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start enforcer
	enf := enforcer.New(cfg, log)
	if err := enf.Start(ctx); err != nil {
		log.WithError(err).Error("Failed to start enforcer")
		os.Exit(1)
	}

	// Start HTTP server
	srv := server.New(cfg.Port, log)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("HTTP server error")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down gracefully")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.WithError(err).Error("HTTP server shutdown error")
	}

	enf.Stop()
	log.Info("Shutdown complete")
}
