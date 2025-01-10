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

	// Log startup info
	logger.Loading("Starting ggquick server...")
	logger.Info("Debug: %v", debug)

	// Check required environment variables
	envVars := map[string]string{
		"GITHUB_TOKEN":   os.Getenv("GITHUB_TOKEN"),
		"OPENAI_API_KEY": os.Getenv("OPENAI_API_KEY"),
	}

	// Log environment status
	for name, value := range envVars {
		if value == "" {
			return fmt.Errorf("required environment variable not set: %s", name)
		}
		logger.Success("Environment variable set: %s", name)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("Received signal: %v", sig)
		cancel()
	}()

	// Initialize server components
	logger.Loading("Initializing server components...")

	aiGen := ai.New(logger)
	logger.Success("AI generator initialized")

	ghClient := github.New(logger)
	logger.Success("GitHub client initialized")

	hooksMgr := hooks.New(logger)
	logger.Success("Hooks manager initialized")

	// Create and start server
	srv, err := server.New(logger, aiGen, ghClient, hooksMgr)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	logger.Success("Server created successfully")

	// Start server
	logger.Loading("Starting HTTP server...")
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	logger.Success("Server shutdown complete")
	return nil
}
