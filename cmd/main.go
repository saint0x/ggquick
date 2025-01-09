package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/saint0x/ggquick/pkg/log"
)

const serverURL = "https://ggquick.fly.dev"

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
		// Verify environment
		if os.Getenv("GITHUB_TOKEN") == "" {
			logger.Error("GITHUB_TOKEN environment variable not set")
			os.Exit(1)
		}

		if os.Getenv("OPENAI_API_KEY") == "" {
			logger.Error("OPENAI_API_KEY environment variable not set")
			os.Exit(1)
		}

		// Check server health
		resp, err := http.Get(serverURL + "/health")
		if err != nil {
			logger.Error("Failed to connect to server: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Error("Server is not healthy (status: %d)", resp.StatusCode)
			os.Exit(1)
		}

		logger.Success("Connected to ggquick ✨")
		logger.Info("Listening for Git events...")
		logger.Info("Press Ctrl+C to stop")

		// Wait for Ctrl+C
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		logger.Info("Shutting down...")

	case "check":
		checkRemoteServer(logger, serverURL)

	default:
		logger.Error("Unknown command: %s", args[0])
		printUsage(logger)
		os.Exit(1)
	}
}

func printUsage(logger *log.Logger) {
	logger.PR("ggquick 🚀")
	logger.Info("AI-powered GitHub PR automation")
	logger.Info("")
	logger.Step("Commands:")
	logger.Success("  start     Start listening for Git events")
	logger.Success("  check     Check server status")
	logger.Info("")
	logger.Step("Options:")
	logger.Info("  --debug    Enable verbose logging")
	logger.Info("")
	logger.Step("Environment:")
	logger.Info("  GITHUB_TOKEN      GitHub token")
	logger.Info("  OPENAI_API_KEY    OpenAI API key")
	logger.Info("")
	logger.Step("Quick Start:")
	logger.Info("  1. Create .env file with:")
	logger.Info("     GITHUB_TOKEN=<token>")
	logger.Info("     OPENAI_API_KEY=<key>")
	logger.Info("  2. Run: ggquick start")
}

func checkRemoteServer(logger *log.Logger, url string) {
	resp, err := http.Get(url + "/health")
	if err != nil {
		logger.Error("Failed to connect to server: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Success("Connected to ggquick ✨")
		logger.Info("Ready to handle Git events")
	} else {
		logger.Error("Server returned status: %d", resp.StatusCode)
		os.Exit(1)
	}
}
