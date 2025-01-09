package main

import (
	"context"
	"flag"
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

var (
	debug = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()
	logger := log.New(*debug)

	// Parse command
	args := flag.Args()
	if len(args) < 1 {
		printUsage(logger)
		os.Exit(1)
	}

	// Handle commands
	switch args[0] {
	case "start":
		startServer(logger)

	case "stop":
		if err := stopServer(logger); err != nil {
			logger.Error("Failed to stop server: %v", err)
			os.Exit(1)
		}

	default:
		logger.Error("Unknown command: %s", args[0])
		printUsage(logger)
		os.Exit(1)
	}
}

func startServer(logger *log.Logger) {
	// Check if server is already running
	pidFile := "/tmp/ggquick.pid"
	if _, err := os.Stat(pidFile); err == nil {
		logger.Error("Server is already running")
		os.Exit(1)
	}

	// Save PID
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		logger.Error("Failed to save PID: %v", err)
		os.Exit(1)
	}

	// Create components
	aiGen := ai.New(logger)
	ghClient := github.New(logger)
	hooksMgr := hooks.New(logger)

	// Create server
	srv, err := server.New(logger, aiGen, ghClient, hooksMgr)
	if err != nil {
		logger.Error("Failed to create server: %v", err)
		os.Exit(1)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal: %v", sig)
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logger.Error("Server error: %v", err)
		os.Remove(pidFile)
		os.Exit(1)
	}
}

func stopServer(logger *log.Logger) error {
	pidFile := "/tmp/ggquick.pid"
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no server running")
		}
		return err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	os.Remove(pidFile)
	logger.Success("Server stopped")
	return nil
}

func printUsage(logger *log.Logger) {
	logger.Info("Usage: ggquick <command>")
	logger.Info("")
	logger.Info("Commands:")
	logger.Info("  start     Start the background server")
	logger.Info("  stop      Stop the server")
	logger.Info("")
	logger.Info("Flags:")
	logger.Info("  --debug   Enable debug logging")
}
