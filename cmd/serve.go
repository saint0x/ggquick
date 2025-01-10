package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/github"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
	"github.com/saint0x/ggquick/pkg/server"
)

func handleServe() error {
	// Initialize logger
	debug := os.Getenv("DEBUG") == "true"
	logger := log.New(debug)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to track if we're already shutting down
	shuttingDown := make(chan struct{}, 1)

	go func() {
		for sig := range sigCh {
			select {
			case <-shuttingDown:
				// Second signal, force exit
				logger.Error("❌ Force stopping...")
				os.Exit(1)
			default:
				// First signal, graceful shutdown
				logger.Info("🛑 Received signal: %v", sig)
				logger.Info("ℹ️ Press Ctrl+C again to force stop")
				shuttingDown <- struct{}{} // Mark that we're shutting down
				cancel()                   // Trigger graceful shutdown
			}
		}
	}()

	// Initialize server components
	aiGen := ai.New(logger)
	ghClient := github.New(logger)
	hooksMgr := hooks.New(logger)

	// Create and start server
	srv, err := server.New(logger, aiGen, ghClient, hooksMgr)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	return nil
}
