package ai

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/saint0x/ggquick/pkg/log"
)

// mockHTTPClient implements HTTPClient for testing
type mockHTTPClient struct {
	response string
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(m.response)),
	}, nil
}

func TestGeneratePRTitle(t *testing.T) {
	logger := log.New(true)
	gen := New(logger)

	// Mock HTTP client
	mockClient := &mockHTTPClient{
		response: `{"choices":[{"message":{"content":"feat(server): Add webhook support"}}]}`,
	}
	gen.SetHTTPClient(mockClient)

	// Test data
	diff := DiffAnalysis{
		Files: []string{"pkg/server/server.go"},
		Changes: map[string]Change{
			"pkg/server/server.go": {
				Path:      "pkg/server/server.go",
				Added:     []string{"func handleWebhook()"},
				Type:      "feature",
				Component: "server",
			},
		},
	}

	// Test PR title generation
	title, err := gen.GeneratePRTitle(diff)
	if err != nil {
		t.Fatalf("Failed to generate PR title: %v", err)
	}

	if title != "feat(server): Add webhook support" {
		t.Errorf("Expected title %q, got %q", "feat(server): Add webhook support", title)
	}
}

func TestGeneratePRDescription(t *testing.T) {
	logger := log.New(true)
	gen := New(logger)

	// Mock HTTP client
	mockClient := &mockHTTPClient{
		response: `{"choices":[{"message":{"content":"Added webhook support to server:\n- New handleWebhook function\n- Support for POST requests"}}]}`,
	}
	gen.SetHTTPClient(mockClient)

	// Test data
	diff := DiffAnalysis{
		Files: []string{"pkg/server/server.go"},
		Changes: map[string]Change{
			"pkg/server/server.go": {
				Path:      "pkg/server/server.go",
				Added:     []string{"func handleWebhook()"},
				Type:      "feature",
				Component: "server",
			},
		},
	}

	// Test PR description generation
	desc, err := gen.GeneratePRDescription(diff)
	if err != nil {
		t.Fatalf("Failed to generate PR description: %v", err)
	}

	expected := "Added webhook support to server:\n- New handleWebhook function\n- Support for POST requests"
	if desc != expected {
		t.Errorf("Expected description %q, got %q", expected, desc)
	}
}

func TestAnalyzeCommit(t *testing.T) {
	logger := log.New(true)
	gen := New(logger)

	// Mock HTTP client
	mockClient := &mockHTTPClient{
		response: `{"choices":[{"message":{"content":"Type: feature\nComponent: server\nDescription: Add webhook support"}}]}`,
	}
	gen.SetHTTPClient(mockClient)

	// Test commit analysis
	analysis, err := gen.AnalyzeCommit("feat(server): Add webhook support")
	if err != nil {
		t.Fatalf("Failed to analyze commit: %v", err)
	}

	if analysis.Type != "feature" {
		t.Errorf("Expected type %q, got %q", "feature", analysis.Type)
	}
	if analysis.Component != "server" {
		t.Errorf("Expected component %q, got %q", "server", analysis.Component)
	}
	if analysis.Description != "Add webhook support" {
		t.Errorf("Expected description %q, got %q", "Add webhook support", analysis.Description)
	}
}
