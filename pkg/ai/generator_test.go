package ai

import (
	"bytes"
	"context"
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

func TestGeneratePR(t *testing.T) {
	logger := log.New(true)
	gen := New(logger)

	// Mock HTTP client
	mockClient := &mockHTTPClient{
		response: `{"choices":[{"message":{"content":"feat(server): Add webhook support\n\nAdded webhook support to server:\n- New handleWebhook function\n- Support for POST requests"}}]}`,
	}
	gen.httpClient = mockClient

	// Test data
	info := RepoInfo{
		Files:         []string{"pkg/server/server.go"},
		BranchName:    "feature/webhook-support",
		CommitMessage: "feat(server): Add webhook support",
		Changes: map[string]Change{
			"pkg/server/server.go": {
				Path:     "pkg/server/server.go",
				Added:    []string{"func handleWebhook()"},
				Modified: []string{"func main()"},
			},
		},
	}

	// Test PR generation
	pr, err := gen.GeneratePR(context.Background(), info)
	if err != nil {
		t.Fatalf("Failed to generate PR: %v", err)
	}

	expectedTitle := "feat(server): Add webhook support"
	if pr.Title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, pr.Title)
	}

	expectedDesc := "Added webhook support to server:\n- New handleWebhook function\n- Support for POST requests"
	if pr.Description != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, pr.Description)
	}
}

func TestGeneratePR_Error(t *testing.T) {
	logger := log.New(true)
	gen := New(logger)

	// Mock HTTP client with error
	mockClient := &mockHTTPClient{
		err: io.ErrUnexpectedEOF,
	}
	gen.httpClient = mockClient

	// Test data
	info := RepoInfo{
		Files:         []string{"pkg/server/server.go"},
		BranchName:    "feature/webhook-support",
		CommitMessage: "feat(server): Add webhook support",
		Changes: map[string]Change{
			"pkg/server/server.go": {
				Path:     "pkg/server/server.go",
				Added:    []string{"func handleWebhook()"},
				Modified: []string{"func main()"},
			},
		},
	}

	// Test PR generation with error
	_, err := gen.GeneratePR(context.Background(), info)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
