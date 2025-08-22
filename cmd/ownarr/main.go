package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keksiqc/ownarr/internal/config"
	"github.com/keksiqc/ownarr/internal/enforcer"
	"github.com/keksiqc/ownarr/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start enforcer
	enf := enforcer.New(cfg)
	if err := enf.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start enforcer: %v\n", err)
		os.Exit(1)
	}

	// Start HTTP server
	srv := server.New(cfg.Port)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("Shutting down gracefully...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP server shutdown error: %v\n", err)
	}

	enf.Stop()
}
