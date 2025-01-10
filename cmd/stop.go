package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/saint0x/ggquick/pkg/log"
)

func handleStop() error {
	logger := log.New(true)
	logger.Loading("🛑 Stopping ggquick server...")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Try to gracefully stop by sending a request
	localBase := fmt.Sprintf("http://localhost:%s", port)
	logger.Loading("🔍 Checking local server on port %s...", port)

	// First try a health check to see if server is running
	resp, err := http.Get(localBase + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		logger.Error("❌ No local server running on port %s", port)
		return nil // Not an error if server isn't running
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Find and kill the process
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", "ggquick.exe")
	default:
		// Find process listening on port
		findCmd := exec.Command("lsof", "-i", fmt.Sprintf(":%s", port))
		output, err := findCmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 { // First line is header
				fields := strings.Fields(lines[1])
				if len(fields) > 1 {
					pid := fields[1]
					cmd = exec.Command("kill", pid)
				}
			}
		}
	}

	if cmd != nil {
		logger.Loading("🔄 Stopping local server process...")
		if err := cmd.Run(); err != nil {
			logger.Error("❌ Failed to stop server process: %v", err)
			return fmt.Errorf("failed to stop server: %w", err)
		}
	}

	// Verify server is stopped by checking health endpoint
	logger.Loading("🔍 Verifying server is stopped...")
	time.Sleep(time.Second) // Give the server a moment to shut down

	resp, err = http.Get(localBase + "/health")
	if err != nil {
		// Error means server is not responding, which is what we want
		logger.Success("✅ Local server stopped successfully")
		return nil
	}
	defer resp.Body.Close()

	// If we can still reach the server, something went wrong
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.Error("❌ Server is still running: %s", string(body))
		return fmt.Errorf("server is still running on port %s", port)
	}

	logger.Success("✅ Local server stopped successfully")
	return nil
}
