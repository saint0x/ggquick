package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/saint0x/ggquick/pkg/log"
)

func handleStart(repoURL string) error {
	logger := log.New(true)
	logger.Loading("Initializing ggquick...")

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

	// Send config to server
	resp, err := http.Post("https://ggquick.fly.dev/config", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send config to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	logger.Success("Configuration saved successfully")
	logger.Info("Repository URL: %s", repoURL)

	return nil
}
