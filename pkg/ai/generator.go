package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/saint0x/ggquick/pkg/log"
)

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// HTTPClient interface for mocking http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Generator handles AI-powered PR content generation
type Generator struct {
	logger     *log.Logger
	cache      sync.Map // Cache for analysis results
	httpClient HTTPClient
}

// Analysis represents code analysis results
type Analysis struct {
	Type        string
	Component   string
	Description string
	CreatedAt   time.Time
}

// DiffAnalysis represents code changes
type DiffAnalysis struct {
	Files     []string          `json:"files"`
	Additions int               `json:"additions"`
	Deletions int               `json:"deletions"`
	Changes   map[string]Change `json:"changes"`
}

// Change represents a file change
type Change struct {
	Path      string   `json:"path"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
	Type      string   `json:"type"` // e.g., "feature", "fix", "refactor"
	Component string   `json:"component"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// New creates a new Generator instance
func New(logger *log.Logger) *Generator {
	return &Generator{
		logger:     logger,
		httpClient: http.DefaultClient,
	}
}

// SetHTTPClient sets a custom HTTP client (useful for testing)
func (g *Generator) SetHTTPClient(client HTTPClient) {
	g.httpClient = client
}

// GeneratePRTitle generates a PR title based on code changes and contributing guidelines
func (g *Generator) GeneratePRTitle(diff DiffAnalysis) (string, error) {
	systemPrompt := "You are a PR title generator. Generate a concise, descriptive title."
	userPrompt := fmt.Sprintf("Generate a PR title for these changes:\n%+v", diff)

	title, err := g.generateWithAI(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(title), nil
}

// GeneratePRDescription generates a PR description based on code changes and contributing guidelines
func (g *Generator) GeneratePRDescription(diff DiffAnalysis) (string, error) {
	systemPrompt := "You are a PR description generator. Generate a clear, detailed description."
	userPrompt := fmt.Sprintf("Generate a PR description for these changes:\n%+v", diff)

	desc, err := g.generateWithAI(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(desc), nil
}

// generateWithAI makes a request to GPT-4
func (g *Generator) generateWithAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	req := openAIRequest{
		Model: "gpt-4",
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		openAIEndpoint,
		bytes.NewBuffer(data),
	)
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

	var aiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return aiResp.Choices[0].Message.Content, nil
}

// AnalyzeCommit analyzes a commit message
func (g *Generator) AnalyzeCommit(msg string) (*Analysis, error) {
	systemPrompt := "Analyze this commit message and extract the type (e.g., feature, fix, refactor) and component."
	systemPrompt += "\nExpect conventional commit format: type(scope): description"

	analysis, err := g.generateWithAI(context.Background(), systemPrompt, msg)
	if err != nil {
		return nil, err
	}

	// Parse AI response
	lines := strings.Split(analysis, "\n")
	result := &Analysis{CreatedAt: time.Now()}

	for _, line := range lines {
		if strings.HasPrefix(line, "Type:") {
			result.Type = strings.TrimSpace(strings.TrimPrefix(line, "Type:"))
		} else if strings.HasPrefix(line, "Component:") {
			result.Component = strings.TrimSpace(strings.TrimPrefix(line, "Component:"))
		} else if strings.HasPrefix(line, "Description:") {
			result.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	return result, nil
}

// AnalyzeCode analyzes code content
func (g *Generator) AnalyzeCode(code string) (*Analysis, error) {
	systemPrompt := "Analyze this code and determine if it's a feature, fix, refactor, etc. Consider error handling, tests, and complexity."

	analysis, err := g.generateWithAI(context.Background(), systemPrompt, code)
	if err != nil {
		return nil, err
	}

	// Parse AI response
	lines := strings.Split(analysis, "\n")
	result := &Analysis{CreatedAt: time.Now()}

	for _, line := range lines {
		if strings.HasPrefix(line, "Type:") {
			result.Type = strings.TrimSpace(strings.TrimPrefix(line, "Type:"))
		} else if strings.HasPrefix(line, "Description:") {
			result.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	return result, nil
}

// CacheAnalysis stores analysis results in cache
func (g *Generator) CacheAnalysis(key string, analysis *Analysis) {
	g.cache.Store(key, analysis)
}

// GetCachedAnalysis retrieves analysis results from cache
func (g *Generator) GetCachedAnalysis(key string) (*Analysis, bool) {
	if value, ok := g.cache.Load(key); ok {
		return value.(*Analysis), true
	}
	return nil, false
}

// IsAnalysisCacheExpired checks if an analysis is expired (older than 24 hours)
func (g *Generator) IsAnalysisCacheExpired(key string) bool {
	if value, ok := g.cache.Load(key); ok {
		analysis := value.(*Analysis)
		return time.Since(analysis.CreatedAt) > 24*time.Hour
	}
	return true
}

// CleanupExpiredCache removes expired entries from the cache
func (g *Generator) CleanupExpiredCache() {
	g.cache.Range(func(key, value interface{}) bool {
		if g.IsAnalysisCacheExpired(key.(string)) {
			g.cache.Delete(key)
		}
		return true
	})
}
