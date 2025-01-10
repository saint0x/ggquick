package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/saint0x/ggquick/pkg/log"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// Generator handles AI-powered PR generation
type Generator struct {
	logger     *log.Logger
	httpClient HTTPClient
	sysPrompt  string
}

// HTTPClient interface for mocking http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New creates a new Generator instance
func New(logger *log.Logger) *Generator {
	g := &Generator{
		logger:     logger,
		httpClient: http.DefaultClient,
	}

	// Load system prompt
	if err := g.loadSystemPrompt(); err != nil {
		logger.Warning("Failed to load system prompt: %v", err)
		g.sysPrompt = "You are an AI assistant helping to generate clear and descriptive pull requests."
	}

	return g
}

// loadSystemPrompt loads the PR generation prompt from sysprompt.json
func (g *Generator) loadSystemPrompt() error {
	data, err := os.ReadFile("sysprompt.json")
	if err != nil {
		return fmt.Errorf("failed to read sysprompt.json: %w", err)
	}

	var prompts map[string]string
	if err := json.Unmarshal(data, &prompts); err != nil {
		return fmt.Errorf("failed to parse sysprompt.json: %w", err)
	}

	if prompt, ok := prompts["pr_description"]; ok {
		g.sysPrompt = prompt
	}
	return nil
}

// GeneratePR generates a PR title and description based on repository information
func (g *Generator) GeneratePR(ctx context.Context, info RepoInfo) (*PRContent, error) {
	// Construct user prompt with all relevant info
	userPrompt := fmt.Sprintf(`Generate a pull request title and description based on the following information:

Branch: %s
Commit Message: %s

Changed Files:
%v

Changes:
%v

`, info.BranchName, info.CommitMessage, info.Files, info.Changes)

	if info.ContributingFile != "" {
		userPrompt += fmt.Sprintf("\nContributing Guidelines:\n%s", info.ContributingFile)
	}

	// Make request to OpenAI
	content, err := g.generateWithAI(ctx, g.sysPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PR: %w", err)
	}

	// Parse response into title and description
	lines := bytes.Split([]byte(content), []byte("\n"))
	pr := &PRContent{}

	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		if pr.Title == "" {
			pr.Title = string(line)
		} else {
			pr.Description = string(bytes.Join(lines[i:], []byte("\n")))
			break
		}
	}

	return pr, nil
}

// generateWithAI makes a request to GPT-4
func (g *Generator) generateWithAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Temperature float64 `json:"temperature"`
	}{
		Model: "gpt-4",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return aiResp.Choices[0].Message.Content, nil
}
