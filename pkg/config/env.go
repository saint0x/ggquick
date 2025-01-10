package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/saint0x/ggquick/pkg/log"
	"github.com/saint0x/ggquick/pkg/openai"
)

// Environment holds validated environment configuration
type Environment struct {
	GitHubToken string
	OpenAIKey   string
	Port        string
	Debug       bool
	FlyAppName  string
}

// Validate checks and validates all required environment variables
func Validate(logger *log.Logger) (*Environment, error) {
	env := &Environment{
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		Port:        os.Getenv("PORT"),
		Debug:       os.Getenv("DEBUG") == "true",
		FlyAppName:  os.Getenv("FLY_APP_NAME"),
	}

	// Validate GitHub token
	if env.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not configured")
	}

	// Validate OpenAI key with a test request
	if env.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}

	// Test OpenAI key
	client := openai.NewClient(env.OpenAIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "Validate token"},
		},
		MaxTokens: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid OPENAI_API_KEY: %w", err)
	}

	// Set default port if not specified
	if env.Port == "" {
		env.Port = "8080"
	}

	return env, nil
}
