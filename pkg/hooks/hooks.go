package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/log"
	"golang.org/x/oauth2"
)

// Manager handles git hooks and GitHub API integration
type Manager struct {
	logger *log.Logger
	github *github.Client
	mu     sync.RWMutex
}

// PullRequestOptions contains options for creating a PR
type PullRequestOptions struct {
	Title       string
	Description string
	Branch      string
	BaseBranch  string
	Labels      []string
}

// Hook represents a git hook
type Hook struct {
	Name     string
	Path     string
	Template string
	Enabled  bool
}

// RepoInfo contains repository information
type RepoInfo struct {
	Path      string
	HooksPath string
}

// New creates a new hooks manager
func New(logger *log.Logger) *Manager {
	return &Manager{
		logger: logger,
		mu:     sync.RWMutex{},
	}
}

// InitGitHub initializes the GitHub client
func (m *Manager) InitGitHub(token string) error {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	m.github = github.NewClient(tc)
	return nil
}

// CreatePullRequest creates a new pull request
func (m *Manager) CreatePullRequest(ctx context.Context, owner, repo string, opts *PullRequestOptions) (*github.PullRequest, error) {
	pr, _, err := m.github.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title:               github.String(opts.Title),
		Body:                github.String(opts.Description),
		Head:                github.String(opts.Branch),
		Base:                github.String(opts.BaseBranch),
		MaintainerCanModify: github.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	// Add labels if specified
	if len(opts.Labels) > 0 {
		_, _, err = m.github.Issues.AddLabelsToIssue(ctx, owner, repo, pr.GetNumber(), opts.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to add labels: %w", err)
		}
	}

	return pr, nil
}

// InstallHooks installs git hooks in the repository
func (m *Manager) InstallHooks(repoPath string) error {
	// Install post-commit hook
	hook := `#!/bin/sh
# ggquick post-commit hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`

	// Write hook file
	if err := writeHook(repoPath, "post-commit", hook); err != nil {
		return fmt.Errorf("failed to install post-commit hook: %w", err)
	}

	// Install post-push hook
	hook = `#!/bin/sh
# ggquick post-push hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`

	if err := writeHook(repoPath, "post-push", hook); err != nil {
		return fmt.Errorf("failed to install post-push hook: %w", err)
	}

	return nil
}

// writeHook writes a git hook file
func writeHook(repoPath, hookName, content string) error {
	hookPath := filepath.Join(repoPath, ".git", "hooks", hookName)
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return err
	}
	return nil
}

// UpdateRepo updates the repository hooks
func (m *Manager) UpdateRepo(info *RepoInfo) error {
	// Validate paths
	if info.Path == "" {
		return fmt.Errorf("repository path is required")
	}

	// Create hooks directory if it doesn't exist
	hooksDir := filepath.Join(info.Path, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Install post-commit hook
	postCommitPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(postCommitPath, []byte(postCommitHook), 0755); err != nil {
		return fmt.Errorf("failed to install post-commit: %w", err)
	}

	// Install post-push hook
	postPushPath := filepath.Join(hooksDir, "post-push")
	if err := os.WriteFile(postPushPath, []byte(postPushHook), 0755); err != nil {
		return fmt.Errorf("failed to install post-push: %w", err)
	}

	return nil
}

const postCommitHook = `#!/bin/sh
# ggquick post-commit hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/webhook" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`

const postPushHook = `#!/bin/sh
# ggquick post-push hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/webhook" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`

// RemoveHooks removes all hooks from a repository
func (m *Manager) RemoveHooks(repoPath string) error {
	// Get hooks directory path
	hooksDir := filepath.Join(repoPath, ".git", "hooks")

	// List of hooks to remove
	hooks := []string{"post-commit", "post-push"}

	// Remove each hook
	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)
		if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", hook, err)
		}
	}

	return nil
}

// ValidateGitRepo validates a git repository
func (m *Manager) ValidateGitRepo(path string) error {
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %w", err)
	}
	return nil
}

// CheckWebhook checks if our webhook already exists for the repository
func (m *Manager) CheckWebhook(ctx context.Context, owner, repo string) (bool, error) {
	// List all hooks
	hooks, _, err := m.github.Repositories.ListHooks(ctx, owner, repo, nil)
	if err != nil {
		return false, fmt.Errorf("failed to list webhooks: %w", err)
	}

	// Check if our webhook exists
	for _, hook := range hooks {
		if url, ok := hook.Config["url"].(string); ok {
			if strings.Contains(url, "ggquick") {
				return true, nil
			}
		}
	}

	return false, nil
}

// CreateHook creates a webhook in the GitHub repository if it doesn't exist
func (m *Manager) CreateHook(ctx context.Context, owner, repo, url string) error {
	// Check if webhook already exists
	exists, err := m.CheckWebhook(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to check webhook: %w", err)
	}

	if exists {
		m.logger.Info("✨ Webhook already exists")
		return nil
	}

	// Create webhook configuration
	config := map[string]interface{}{
		"url":          url,
		"content_type": "json",
		"insecure_ssl": "0",
	}

	// Create webhook
	hook := &github.Hook{
		Config: config,
		Events: []string{"push"},
		Active: github.Bool(true),
	}

	// Call GitHub API to create webhook
	_, _, err = m.github.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	m.logger.Success("✅ Created new webhook")
	return nil
}

// DeleteHook deletes the webhook from the GitHub repository
func (m *Manager) DeleteHook(ctx context.Context, owner, repo string) error {
	// List all hooks
	hooks, _, err := m.github.Repositories.ListHooks(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	// Find and delete our webhook
	for _, hook := range hooks {
		if url, ok := hook.Config["url"].(string); ok && strings.Contains(url, "ggquick") {
			_, err := m.github.Repositories.DeleteHook(ctx, owner, repo, *hook.ID)
			if err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("webhook not found")
}
