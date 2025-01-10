package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/saint0x/ggquick/pkg/log"
)

// checkHealth checks if the server is healthy
func checkHealth(logger *log.Logger, baseURL string) error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy: status %d", resp.StatusCode)
	}

	logger.Success("✅ Server is healthy")
	return nil
}

func handleStart(repoURL string) error {
	logger := log.New(true)
	logger.Loading("🚀 Initializing ggquick client...")
	logger.Info("📝 Target repository: %s", repoURL)

	// Validate repository URL
	if repoURL == "" {
		return fmt.Errorf("repository URL is required")
	}

	// Create config
	config := struct {
		RepoURL string `json:"repo_url"`
	}{
		RepoURL: repoURL,
	}

	// Marshal config
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Try remote server first (fly.io)
	remoteBase := "https://ggquick.fly.dev"

	// For local server, use the port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	localBase := fmt.Sprintf("http://localhost:%s", port)

	// Check remote server health first
	logger.Loading("🔍 Checking remote server (ggquick.fly.dev)...")
	if err := checkHealth(logger, remoteBase); err == nil {
		// Remote server is healthy, send config
		logger.Loading("📤 Sending configuration to remote server...")
		resp, err := http.Post(remoteBase+"/config", "application/json", bytes.NewBuffer(data))
		if err != nil {
			logger.Error("❌ Failed to send configuration to remote server: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return handleResponse(logger, resp, "remote")
			}
			// Read error response
			body, _ := io.ReadAll(resp.Body)
			logger.Error("❌ Server returned error: %s", string(body))
		}
	} else {
		logger.Error("❌ Remote server health check failed: %v", err)
	}

	// If remote server failed, try local server
	logger.Info("ℹ️ Remote server unavailable, falling back to local server...")
	logger.Info("🔍 Checking local server on port %s...", port)
	if err := checkHealth(logger, localBase); err != nil {
		logger.Error("❌ Local server health check failed: %v", err)
		return fmt.Errorf("❌ both remote and local servers are unavailable")
	}

	// Send config to local server
	logger.Loading("📤 Sending configuration to local server...")
	resp, err := http.Post(localBase+"/config", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send config to server: %w", err)
	}
	defer resp.Body.Close()

	return handleResponse(logger, resp, "local")
}

// handleResponse processes the server response and logs the result
func handleResponse(logger *log.Logger, resp *http.Response, serverType string) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Parse server response to confirm config was stored
	var response struct {
		Status string `json:"status"`
		Owner  string `json:"owner"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse server response: %w", err)
	}

	if response.Status != "config_stored" {
		return fmt.Errorf("server did not confirm config storage, got status: %s", response.Status)
	}

	// Single, clear success sequence
	logger.Success("✨ Configuration sent successfully to %s server", serverType)
	logger.Success("✅ Server confirmed configuration is stored")
	logger.Info("📦 Repository configured: %s/%s", response.Owner, response.Name)
	logger.Success("🔄 Ready to process Git events")

	return nil
}
