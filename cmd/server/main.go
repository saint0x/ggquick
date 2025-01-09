package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/github"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
	"github.com/saint0x/ggquick/pkg/server"
)

func main() {
	// Initialize logger
	debug := os.Getenv("DEBUG") == "true"
	logger := log.New(debug)

	// Log startup info
	logger.Loading("Starting ggquick server...")
	logger.Info("App: %s", os.Getenv("FLY_APP_NAME"))
	logger.Info("Debug: %v", debug)
	logger.Info("Port: %s", os.Getenv("PORT"))

	// Check environment variables
	envVars := map[string]string{
		"GITHUB_TOKEN":   os.Getenv("GITHUB_TOKEN"),
		"OPENAI_API_KEY": os.Getenv("OPENAI_API_KEY"),
		"PORT":           os.Getenv("PORT"),
		"BIND":           os.Getenv("BIND"),
	}

	// Log environment status
	for name, value := range envVars {
		if value == "" {
			logger.Error("Required environment variable not set: %s", name)
			os.Exit(1)
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
		logger.Error("Failed to create server: %v", err)
		os.Exit(1)
	}
	logger.Success("Server created successfully")

	// Start server
	logger.Loading("Starting HTTP server on %s...", envVars["BIND"])
	if err := srv.Start(ctx); err != nil {
		logger.Error("Server error: %v", err)
		os.Exit(1)
	}

	// Wait for shutdown
	<-ctx.Done()
	logger.Success("Server shutdown complete")
}
