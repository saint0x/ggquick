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

	// Single, clear startup sequence
	logger.Loading("🚀 Starting ggquick server...")
	logger.Info("🔧 Debug mode: %v", debug)

	// Check environment
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		logger.Success("✅ GITHUB_TOKEN configured")
	} else {
		logger.Error("❌ GITHUB_TOKEN not configured")
		os.Exit(1)
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		logger.Success("✅ OPENAI_API_KEY configured")
	} else {
		logger.Error("❌ OPENAI_API_KEY not configured")
		os.Exit(1)
	}

	// Initialize components
	logger.Loading("⚙️ Initializing components...")

	aiGen := ai.New(logger)
	logger.Success("✅ AI generator ready")

	ghClient := github.New(logger)
	logger.Success("✅ GitHub client ready")

	hooksMgr := hooks.New(logger)
	logger.Success("✅ Git hooks ready")

	// Create and start server
	srv, err := server.New(logger, aiGen, ghClient, hooksMgr)
	if err != nil {
		logger.Error("❌ Failed to create server: %v", err)
		os.Exit(1)
	}
	logger.Success("✅ Server initialized")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("🛑 Received signal: %v", sig)
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logger.Error("❌ Server error: %v", err)
		os.Exit(1)
	}

	// Wait for shutdown
	<-ctx.Done()
	logger.Success("✨ Server shutdown complete")
}
