package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
)

// mockGenerator implements AIGenerator
type mockGenerator struct {
	prContent *ai.PRContent
	err       error
}

func (m *mockGenerator) GeneratePR(_ context.Context, _ ai.RepoInfo) (*ai.PRContent, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.prContent == nil {
		return &ai.PRContent{
			Title:       "test PR title",
			Description: "test PR description",
		}, nil
	}
	return m.prContent, nil
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

func setupTestServer(t *testing.T) (*Server, *mockGenerator, *mockClient, *mockManager, func()) {
	logger := log.New(true)

	mockGen := &mockGenerator{
		prContent: &ai.PRContent{
			Title:       "test PR title",
			Description: "test PR description",
		},
	}

	mockGH := &mockClient{
		getDefaultBranchFunc: func(context.Context, string, string) (string, error) {
			return "main", nil
		},
		getDiffFunc: func(context.Context, string, string, string, string) (string, error) {
			return "test diff", nil
		},
		getCommitMessageFunc: func(context.Context, string, string, string) (string, error) {
			return "test commit message", nil
		},
	}

	mockHooks := &mockManager{}

	srv, err := New(logger, mockGen, mockGH, mockHooks)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	cleanup := func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}

	return srv, mockGen, mockGH, mockHooks, cleanup
}

func TestWebhookHandling(t *testing.T) {
	srv, _, _, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create test request
	payload := struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	}{
		Ref: "refs/heads/feature/test",
		SHA: "abc123",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest("POST", "/push", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	// Handle request
	srv.handlePush(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("Expected body to contain 'ok', got %q", rec.Body.String())
	}
}

// Add more test cases for error scenarios
func TestWebhookHandling_Errors(t *testing.T) {
	srv, mockGen, _, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Test case: AI error
	mockGen.err = fmt.Errorf("AI error")

	payload := struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	}{
		Ref: "refs/heads/feature/test",
		SHA: "abc123",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest("POST", "/push", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	srv.handlePush(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
