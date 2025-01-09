package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	case "check":
		if err := checkServer(logger); err != nil {
			logger.Error("Server status check failed: %v", err)
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
	if pid, err := checkPID(pidFile); err == nil {
		logger.Error("Server is already running (PID: %d)", pid)
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

	// Handle signals in a separate goroutine
	go func() {
		sig := <-sigCh
		if logger.IsDebug() {
			logger.Info("Received signal: %v", sig)
		}
		logger.Info("Shutting down server...")
		cancel()

		// Wait for a second signal for force quit
		select {
		case <-sigCh:
			logger.Warning("Forced shutdown")
			os.Remove(pidFile)
			os.Exit(1)
		case <-time.After(5 * time.Second):
			logger.Error("Server took too long to stop")
			os.Remove(pidFile)
			os.Exit(1)
		}
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logger.Error("Server error: %v", err)
		os.Remove(pidFile)
		os.Exit(1)
	}

	// Clean up
	os.Remove(pidFile)
	os.Exit(0)
}

func stopServer(logger *log.Logger) error {
	pidFile := "/tmp/ggquick.pid"
	pid, err := checkPID(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warning("No server is currently running")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err.Error() == "os: process already finished" {
			logger.Warning("Server process was not running")
			os.Remove(pidFile)
			return nil
		}
		return fmt.Errorf("failed to stop server: %w", err)
	}

	os.Remove(pidFile)
	logger.Success("Server stopped successfully (PID: %d)", pid)
	return nil
}

func checkServer(logger *log.Logger) error {
	pidFile := "/tmp/ggquick.pid"
	pid, err := checkPID(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warning("No server is running")
			return nil
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		logger.Warning("No server is running")
		return nil
	}

	// Check if process responds
	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFile)
		logger.Warning("No server is running")
		return nil
	}

	// Read port
	portFile := fmt.Sprintf("%s/.ggquick/port", os.Getenv("HOME"))
	portBytes, err := os.ReadFile(portFile)
	if err != nil {
		return fmt.Errorf("failed to read port file: %w", err)
	}
	port := string(portBytes)

	logger.Success("Server is running:")
	logger.Info("- Process ID: %d", pid)
	logger.Info("- Port: %s", port)
	logger.Info("- Webhook URL: http://localhost:%s/push", port)
	return nil
}

func checkPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content")
	}

	return pid, nil
}

func printUsage(logger *log.Logger) {
	logger.PR("ggquick 🚀")
	logger.Info("AI-powered GitHub PR automation")
	logger.Info("")
	logger.Step("Commands:")
	logger.Success("  start     Start the automation server")
	logger.Success("  stop      Stop the server")
	logger.Success("  check     Show server status")
	logger.Info("")
	logger.Step("Options:")
	logger.Info("  --debug    Enable verbose logging")
	logger.Info("")
	logger.Step("Environment:")
	logger.Info("  GITHUB_TOKEN         GitHub token")
	logger.Info("  GITHUB_REPOSITORY    Target repository (e.g., user/repo)")
	logger.Info("  GGQUICK_PORT        Server port (default: 8080)")
	logger.Info("")
	logger.Step("Quick Start:")
	logger.Info("  export GITHUB_TOKEN=<token>")
	logger.Info("  export GITHUB_REPOSITORY=user/repo")
	logger.Info("  ggquick start")
}
