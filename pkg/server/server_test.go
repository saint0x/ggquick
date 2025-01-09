package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

// mockClient implements GitHubClient for testing
type mockClient struct {
	createPRFunc         func(context.Context, string, string, string, string, string, string) (*gogithub.PullRequest, error)
	getDefaultBranchFunc func(context.Context, string, string) (string, error)
	parseRepoURLFunc     func(string) (string, string, error)
	getContributingFunc  func(context.Context, string, string) (string, error)
	getBranchesFunc      func(context.Context, string, string) ([]*gogithub.Branch, error)
	getPRsFunc           func(context.Context, string, string, int) ([]*gogithub.PullRequest, error)
	getDiffFunc          func(context.Context, string, string, string, string) (string, error)
	getCommitMessageFunc func(context.Context, string, string, string) (string, error)
	prCreated            bool
}

func (m *mockClient) CreatePR(ctx context.Context, owner, repo, title, body, head, base string) (*gogithub.PullRequest, error) {
	if m.createPRFunc != nil {
		return m.createPRFunc(ctx, owner, repo, title, body, head, base)
	}
	return nil, nil
}

func (m *mockClient) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	if m.getDefaultBranchFunc != nil {
		return m.getDefaultBranchFunc(ctx, owner, repo)
	}
	return "main", nil
}

func (m *mockClient) ParseRepoURL(url string) (string, string, error) {
	if m.parseRepoURLFunc != nil {
		return m.parseRepoURLFunc(url)
	}
	return "owner", "repo", nil
}

func (m *mockClient) GetContributingGuide(ctx context.Context, owner, repo string) (string, error) {
	if m.getContributingFunc != nil {
		return m.getContributingFunc(ctx, owner, repo)
	}
	return "", nil
}

func (m *mockClient) GetBranches(ctx context.Context, owner, repo string) ([]*gogithub.Branch, error) {
	if m.getBranchesFunc != nil {
		return m.getBranchesFunc(ctx, owner, repo)
	}
	return nil, nil
}

func (m *mockClient) GetPRs(ctx context.Context, owner, repo string, limit int) ([]*gogithub.PullRequest, error) {
	if m.getPRsFunc != nil {
		return m.getPRsFunc(ctx, owner, repo, limit)
	}
	return nil, nil
}

func (m *mockClient) GetDiff(ctx context.Context, owner, repo, base, head string) (string, error) {
	if m.getDiffFunc != nil {
		return m.getDiffFunc(ctx, owner, repo, base, head)
	}
	return "", nil
}

func (m *mockClient) GetCommitMessage(ctx context.Context, owner, repo, sha string) (string, error) {
	if m.getCommitMessageFunc != nil {
		return m.getCommitMessageFunc(ctx, owner, repo, sha)
	}
	return "test commit message", nil
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
	mockGH := &mockClient{}
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

func TestServer_handlePush(t *testing.T) {
	mockGen := &mockGenerator{
		titleResp: "Test PR Title",
		descResp:  "Test PR Description",
	}
	mockGH := &mockClient{
		createPRFunc: func(ctx context.Context, owner, repo, title, body, head, base string) (*gogithub.PullRequest, error) {
			return &gogithub.PullRequest{}, nil
		},
		getDefaultBranchFunc: func(ctx context.Context, owner, repo string) (string, error) {
			return "main", nil
		},
	}
	mockHooks := &mockManager{}

	srv, err := New(log.New(true), mockGen, mockGH, mockHooks)
	if err != nil {
		t.Fatal("Failed to create server:", err)
	}

	// Create test request
	body := strings.NewReader(`{"ref": "refs/heads/test-branch"}`)
	req := httptest.NewRequest(http.MethodPost, "/push", body)
	w := httptest.NewRecorder()

	// Handle request
	srv.handlePush(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}
