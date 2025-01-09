package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
)

// mockGenerator implements AIGenerator
type mockGenerator struct {
	titleResp string
	descResp  string
}

func (m *mockGenerator) GeneratePRTitle(_ ai.DiffAnalysis) (string, error) {
	return m.titleResp, nil
}

func (m *mockGenerator) GeneratePRDescription(_ ai.DiffAnalysis) (string, error) {
	return m.descResp, nil
}

func (m *mockGenerator) AnalyzeCommit(string) (*ai.Analysis, error) {
	return &ai.Analysis{}, nil
}

func (m *mockGenerator) AnalyzeCode(string) (*ai.Analysis, error) {
	return &ai.Analysis{}, nil
}

// mockClient implements GitHubClient
type mockClient struct {
	prCreated     bool
	prTitle       string
	prBody        string
	prHead        string
	prBase        string
	defaultBranch string
}

func (m *mockClient) CreatePR(_ context.Context, _, _, title, body, head, base string) (*gogithub.PullRequest, error) {
	m.prCreated = true
	m.prTitle = title
	m.prBody = body
	m.prHead = head
	m.prBase = base
	return &gogithub.PullRequest{}, nil
}

func (m *mockClient) GetDefaultBranch(_ context.Context, _, _ string) (string, error) {
	return m.defaultBranch, nil
}

func (m *mockClient) ParseRepoURL(url string) (owner, repo string, err error) {
	return "test-owner", "test-repo", nil
}

func (m *mockClient) GetContributingGuide(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (m *mockClient) GetBranches(_ context.Context, _, _ string) ([]*gogithub.Branch, error) {
	return nil, nil
}

func (m *mockClient) GetPRs(_ context.Context, _, _ string, _ int) ([]*gogithub.PullRequest, error) {
	return nil, nil
}

func (m *mockClient) GetDiff(_ context.Context, _, _, _, _ string) (string, error) {
	return "", nil
}

// mockManager implements HooksManager
type mockManager struct{}

func (m *mockManager) InstallHooks(string) error       { return nil }
func (m *mockManager) InitGitHub(_, _, _ string) error { return nil }
func (m *mockManager) CreatePullRequest(_ context.Context, _ *hooks.PullRequestOptions) (*gogithub.PullRequest, error) {
	return nil, nil
}
func (m *mockManager) UpdateRepo(_ *hooks.RepoInfo) error { return nil }
func (m *mockManager) RemoveHooks(string) error           { return nil }
func (m *mockManager) ValidateGitRepo(string) error       { return nil }

func setupTestServer(t *testing.T) (*Server, *mockGenerator, *mockClient, func()) {
	logger := log.New(true)
	mockGen := &mockGenerator{
		titleResp: "Test PR Title",
		descResp:  "Test PR Description",
	}
	mockGH := &mockClient{
		defaultBranch: "main",
	}
	mockHooks := &mockManager{}

	s, err := New(logger, mockGen, mockGH, mockHooks)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	return s, mockGen, mockGH, func() {
		s.Stop()
	}
}

func TestWebhookHandling(t *testing.T) {
	// Set required environment variable
	os.Setenv("GITHUB_REPOSITORY", "test-owner/test-repo")
	defer os.Unsetenv("GITHUB_REPOSITORY")

	tests := []struct {
		name           string
		method         string
		payload        map[string]string
		contentType    string
		expectedStatus int
		checkPR        bool
	}{
		{
			name:   "valid json webhook",
			method: "POST",
			payload: map[string]string{
				"ref":    "refs/heads/feature-branch",
				"after":  "abc123",
				"before": "def456",
			},
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			checkPR:        true,
		},
		{
			name:   "valid text webhook",
			method: "POST",
			payload: map[string]string{
				"sha":     "abc123",
				"message": "test commit",
				"author":  "test-author",
			},
			contentType:    "text/plain",
			expectedStatus: http.StatusOK,
			checkPR:        true,
		},
		{
			name:           "invalid method",
			method:         "GET",
			payload:        nil,
			expectedStatus: http.StatusMethodNotAllowed,
			checkPR:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, mockGH, cleanup := setupTestServer(t)
			defer cleanup()

			var body []byte
			var err error
			if tt.payload != nil {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/push", bytes.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			s.handlePush(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Wait for PR creation to complete
			if tt.checkPR {
				time.Sleep(100 * time.Millisecond)
				if !mockGH.prCreated {
					t.Error("Expected PR to be created")
				}
			}
		})
	}
}
