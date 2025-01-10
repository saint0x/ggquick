package ai

import (
	"context"
	"fmt"

	"github.com/saint0x/ggquick/pkg/log"
	"github.com/saint0x/ggquick/pkg/openai"
)

// Generator handles AI operations
type Generator struct {
	logger *log.Logger
	client *openai.Client
}

// New creates a new AI generator
func New(logger *log.Logger) *Generator {
	if logger == nil {
		return nil
	}

	return &Generator{
		logger: logger,
	}
}

// Initialize sets up the OpenAI client with a validated key
func (g *Generator) Initialize(key string) error {
	client := openai.NewClient(key)
	g.client = client
	return nil
}

// GeneratePR generates a pull request description
func (g *Generator) GeneratePR(ctx context.Context, info RepoInfo) (*PRContent, error) {
	// Create chat completion request
	messages := []openai.ChatCompletionMessage{
		{
			Role: "system",
			Content: `You are a helpful AI that generates clear and concise pull request descriptions.
Focus on explaining the changes and their impact. Be professional but conversational.`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf("Generate a PR description for branch '%s' with commit message: %s",
				info.BranchName, info.CommitMessage),
		},
	}

	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PR: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	// Extract title and description
	content := resp.Choices[0].Message.Content
	title := info.CommitMessage // Use commit message as title for now
	description := content

	return &PRContent{
		Title:       title,
		Description: description,
	}, nil
}
